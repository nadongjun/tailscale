// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package health

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"tailscale.com/version"
)

func TestAppendWarnableDebugFlags(t *testing.T) {
	var tr Tracker

	for i := range 10 {
		w := Register(&Warnable{
			Code:         WarnableCode(fmt.Sprintf("warnable-code-%d", i)),
			MapDebugFlag: fmt.Sprint(i),
		})
		defer unregister(w)
		if i%2 == 0 {
			tr.SetUnhealthy(w, Args{"test-arg": fmt.Sprint(i)})
		}
	}

	want := []string{"z", "y", "0", "2", "4", "6", "8"}

	var got []string
	for range 20 {
		got = append(got[:0], "z", "y")
		got = tr.AppendWarnableDebugFlags(got)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("AppendWarnableDebugFlags = %q; want %q", got, want)
		}
	}
}

// Test that all exported methods on *Tracker don't panic with a nil receiver.
func TestNilMethodsDontCrash(t *testing.T) {
	var nilt *Tracker
	rv := reflect.ValueOf(nilt)
	for i := 0; i < rv.NumMethod(); i++ {
		mt := rv.Type().Method(i)
		t.Logf("calling Tracker.%s ...", mt.Name)
		var args []reflect.Value
		for j := 0; j < mt.Type.NumIn(); j++ {
			if j == 0 && mt.Type.In(j) == reflect.TypeFor[*Tracker]() {
				continue
			}
			args = append(args, reflect.Zero(mt.Type.In(j)))
		}
		rv.Method(i).Call(args)
	}
}

func TestSetUnhealthyWithDuplicateThenHealthyAgain(t *testing.T) {
	ht := Tracker{}
	if len(ht.Strings()) != 0 {
		t.Fatalf("before first insertion, len(newTracker.Strings) = %d; want = 0", len(ht.Strings()))
	}

	ht.SetUnhealthy(testWarnable, Args{ArgError: "Hello world 1"})
	want := []string{"Hello world 1"}
	if !reflect.DeepEqual(ht.Strings(), want) {
		t.Fatalf("after calling SetUnhealthy, newTracker.Strings() = %v; want = %v", ht.Strings(), want)
	}

	// Adding a second warning state with the same WarningCode overwrites the existing warning state,
	// the count shouldn't have changed.
	ht.SetUnhealthy(testWarnable, Args{ArgError: "Hello world 2"})
	want = []string{"Hello world 2"}
	if !reflect.DeepEqual(ht.Strings(), want) {
		t.Fatalf("after insertion of same WarningCode, newTracker.Strings() = %v; want = %v", ht.Strings(), want)
	}

	ht.SetHealthy(testWarnable)
	want = []string{}
	if !reflect.DeepEqual(ht.Strings(), want) {
		t.Fatalf("after setting the healthy, newTracker.Strings() = %v; want = %v", ht.Strings(), want)
	}
}

func TestRemoveAllWarnings(t *testing.T) {
	ht := Tracker{}
	if len(ht.Strings()) != 0 {
		t.Fatalf("before first insertion, len(newTracker.Strings) = %d; want = 0", len(ht.Strings()))
	}

	ht.SetUnhealthy(testWarnable, Args{"Text": "Hello world 1"})
	if len(ht.Strings()) != 1 {
		t.Fatalf("after first insertion, len(newTracker.Strings) = %d; want = %d", len(ht.Strings()), 1)
	}

	ht.SetHealthy(testWarnable)
	if len(ht.Strings()) != 0 {
		t.Fatalf("after RemoveAll, len(newTracker.Strings) = %d; want = 0", len(ht.Strings()))
	}
}

// TestWatcher tests that a registered watcher function gets called with the correct
// Warnable and non-nil/nil UnhealthyState upon setting a Warnable to unhealthy/healthy.
func TestWatcher(t *testing.T) {
	ht := Tracker{}
	wantText := "Hello world"
	becameUnhealthy := make(chan struct{})
	becameHealthy := make(chan struct{})

	watcherFunc := func(w *Warnable, us *UnhealthyState) {
		if w != testWarnable {
			t.Fatalf("watcherFunc was called, but with an unexpected Warnable: %v, want: %v", w, testWarnable)
		}

		if us != nil {
			if us.Text != wantText {
				t.Fatalf("unexpected us.Text: %s, want: %s", us.Text, wantText)
			}
			if us.Args[ArgError] != wantText {
				t.Fatalf("unexpected us.Args[ArgError]: %s, want: %s", us.Args[ArgError], wantText)
			}
			becameUnhealthy <- struct{}{}
		} else {
			becameHealthy <- struct{}{}
		}
	}

	unregisterFunc := ht.RegisterWatcher(watcherFunc)
	if len(ht.watchers) != 1 {
		t.Fatalf("after RegisterWatcher, len(newTracker.watchers) = %d; want = 1", len(ht.watchers))
	}
	ht.SetUnhealthy(testWarnable, Args{ArgError: wantText})

	select {
	case <-becameUnhealthy:
		// Test passed because the watcher got notified of an unhealthy state
	case <-becameHealthy:
		// Test failed because the watcher got of a healthy state instead of an unhealthy one
		t.Fatalf("watcherFunc was called with a healthy state")
	case <-time.After(1 * time.Second):
		t.Fatalf("watcherFunc didn't get called upon calling SetUnhealthy")
	}

	ht.SetHealthy(testWarnable)

	select {
	case <-becameUnhealthy:
		// Test failed because the watcher got of an unhealthy state instead of a healthy one
		t.Fatalf("watcherFunc was called with an unhealthy state")
	case <-becameHealthy:
		// Test passed because the watcher got notified of a healthy state
	case <-time.After(1 * time.Second):
		t.Fatalf("watcherFunc didn't get called upon calling SetUnhealthy")
	}

	unregisterFunc()
	if len(ht.watchers) != 0 {
		t.Fatalf("after unregisterFunc, len(newTracker.watchers) = %d; want = 0", len(ht.watchers))
	}
}

func TestRegisterWarnablePanicsWithDuplicate(t *testing.T) {
	w := &Warnable{
		Code: "test-warnable-1",
	}

	Register(w)
	defer unregister(w)
	if registeredWarnables[w.Code] != w {
		t.Fatalf("after Register, registeredWarnables[%s] = %v; want = %v", w.Code, registeredWarnables[w.Code], w)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Registering the same Warnable twice didn't panic")
		}
	}()
	Register(w)
}

func TestIgnoresSetUnhealthyDuringStartup(t *testing.T) {
	testWarnable.IgnoredDuringStartup = true
	ht := Tracker{}
	ht.SetIPNState("Starting", true)

	var want []WarnableCode
	if version.IsUnstableBuild() {
		want = []WarnableCode{unstableWarnable.Code}
	} else {
		want = []WarnableCode{}
	}

	if len(ht.CurrentState().Warnings) != len(want) {
		t.Fatalf("after SetIPNState, len(newTracker.CurrentState().Warnings) = %d; want = %d", len(ht.CurrentState().Warnings), len(want))
	}

	ht.SetUnhealthy(testWarnable, Args{ArgError: "Hello world 1"})
	if len(ht.CurrentState().Warnings) != len(want) {
		t.Fatalf("after SetUnhealthy, len(newTracker.CurrentState().Warnings) = %d; want = %d", len(ht.CurrentState().Warnings), len(want))
	}

	// advance time by 6 seconds to pretend the startup period ended
	ht.ipnWantRunningSetTime = time.Now().Add(-time.Second * 6)
	ht.SetUnhealthy(testWarnable, Args{ArgError: "Hello world 1"})
	if len(ht.CurrentState().Warnings) != len(want)+1 {
		t.Fatalf("after SetUnhealthy, len(newTracker.CurrentState().Warnings) = %d; want = %d", len(ht.CurrentState().Warnings), len(want))
	}

	testWarnable.IgnoredDuringStartup = false
}
