ARG BASE
FROM ${BASE}

ARG BASE

RUN echo "Install openssh, needed for scp."
RUN if echo "$BASE" | grep "ubuntu:"; then apt-get update -y && apt-get install -y openssh-client; fi
RUN if echo "$BASE" | grep "alpine:"; then apk add openssh; fi

# Note - on Ubuntu, we do not create the user's home directory, pam_mkhomedir will do that
# for us, and we want to test that PAM gets triggered by Tailscale SSH.
RUN if echo "$BASE" | grep "ubuntu:"; then groupadd -g 10000 groupone && groupadd -g 10001 grouptwo && useradd -g 10000 -G 10001 -u 10002 testuser; fi
# On Alpine, we can't configure pam_mkhomdir, so go ahead and create home directory.
RUN if echo "$BASE" | grep "alpine:"; then addgroup -g 10000 groupone && addgroup -g 10001 grouptwo && adduser -u 10002 -D testuser && addgroup testuser groupone && addgroup testuser grouptwo; fi

RUN echo "Set up pam_mkhomedir."
RUN if echo "$BASE" | grep "ubuntu:"; then sed -i -e 's/Default: no/Default: yes/g' /usr/share/pam-configs/mkhomedir; fi
RUN if echo "$BASE" | grep "ubuntu:"; then cat /usr/share/pam-configs/mkhomedir; fi
RUN if echo "$BASE" | grep "ubuntu:"; then pam-auth-update --enable mkhomedir; fi

COPY tailscaled .
COPY tailssh.test .

RUN chmod 755 tailscaled

# RUN echo "First run tests normally."
RUN eval `ssh-agent -s` && TAILSCALED_PATH=`pwd`tailscaled ./tailssh.test -test.v -test.run TestSSHAgentForwarding
RUN if echo "$BASE" | grep "ubuntu:"; then rm -Rf /home/testuser; fi
RUN TAILSCALED_PATH=`pwd`tailscaled ./tailssh.test -test.v -test.run TestIntegrationSFTP
RUN if echo "$BASE" | grep "ubuntu:"; then rm -Rf /home/testuser; fi
RUN TAILSCALED_PATH=`pwd`tailscaled ./tailssh.test -test.v -test.run TestIntegrationSCP
RUN if echo "$BASE" | grep "ubuntu:"; then rm -Rf /home/testuser; fi
RUN TAILSCALED_PATH=`pwd`tailscaled ./tailssh.test -test.v -test.run TestIntegrationSSH

RUN echo "Then run tests as non-root user testuser and make sure tests still pass."
RUN chown testuser:groupone /tmp/tailscalessh.log
RUN TAILSCALED_PATH=`pwd`tailscaled eval `su -m testuser -c ssh-agent -s` && su -m testuser -c "./tailssh.test -test.v -test.run TestSSHAgentForwarding"
RUN TAILSCALED_PATH=`pwd`tailscaled su -m testuser -c "./tailssh.test -test.v -test.run TestIntegration TestDoDropPrivileges"
RUN chown root:root /tmp/tailscalessh.log

RUN if echo "$BASE" | grep "ubuntu:"; then echo "Then run tests in a system that's pretending to be SELinux in enforcing mode"; fi
# Remove execute permissions for /usr/bin/login so that it fails.
RUN if echo "$BASE" | grep "ubuntu:"; then mv /usr/bin/login /tmp/login_orig; fi
# Use nonsense for /usr/bin/login so that it fails.
# It's not the same failure mode as in SELinux, but failure is good enough for test.
RUN if echo "$BASE" | grep "ubuntu:"; then echo "adsfasdfasdf" > /usr/bin/login; fi
RUN if echo "$BASE" | grep "ubuntu:"; then chmod 755 /usr/bin/login; fi
# Simulate getenforce command
RUN if echo "$BASE" | grep "ubuntu:"; then printf "#!/bin/bash\necho 'Enforcing'" > /usr/bin/getenforce; fi
RUN if echo "$BASE" | grep "ubuntu:"; then chmod 755 /usr/bin/getenforce; fi
RUN if echo "$BASE" | grep "ubuntu:"; then eval `ssh-agent -s` && TAILSCALED_PATH=`pwd`tailscaled ./tailssh.test -test.v -test.run TestSSHAgentForwarding; fi
RUN if echo "$BASE" | grep "ubuntu:"; then TAILSCALED_PATH=`pwd`tailscaled ./tailssh.test -test.v -test.run TestIntegration; fi
RUN if echo "$BASE" | grep "ubuntu:"; then mv /tmp/login_orig /usr/bin/login; fi
RUN if echo "$BASE" | grep "ubuntu:"; then rm /usr/bin/getenforce; fi

RUN echo "Then remove the login command and make sure tests still pass."
RUN rm `which login`
RUN eval `ssh-agent -s` && TAILSCALED_PATH=`pwd`tailscaled ./tailssh.test -test.v -test.run TestSSHAgentForwarding
RUN if echo "$BASE" | grep "ubuntu:"; then rm -Rf /home/testuser; fi
RUN TAILSCALED_PATH=`pwd`tailscaled ./tailssh.test -test.v -test.run TestIntegrationSFTP
RUN if echo "$BASE" | grep "ubuntu:"; then rm -Rf /home/testuser; fi
RUN TAILSCALED_PATH=`pwd`tailscaled ./tailssh.test -test.v -test.run TestIntegrationSCP
RUN if echo "$BASE" | grep "ubuntu:"; then rm -Rf /home/testuser; fi
RUN TAILSCALED_PATH=`pwd`tailscaled ./tailssh.test -test.v -test.run TestIntegrationSSH

RUN echo "Then remove the su command and make sure tests still pass."
RUN chown root:root /tmp/tailscalessh.log
RUN rm `which su`
RUN eval `ssh-agent -s` && TAILSCALED_PATH=`pwd`tailscaled ./tailssh.test -test.v -test.run TestSSHAgentForwarding
RUN TAILSCALED_PATH=`pwd`tailscaled ./tailssh.test -test.v -test.run TestIntegration

RUN echo "Test doDropPrivileges"
RUN TAILSCALED_PATH=`pwd`tailscaled ./tailssh.test -test.v -test.run TestDoDropPrivileges
