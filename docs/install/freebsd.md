### Install on FreeBSD

Create gitlab-runner user and group:

```bash
sudo pw group add -n gitlab-runner
sudo pw user add -n gitlab-runner -g gitlab-runner -s /usr/local/bin/bash
sudo mkdir /home/gitlab-runner
sudo chown gitlab-runner:gitlab-runner /home/gitlab-runner
```

Download the binary for your system:

```bash
 sudo wget -O /usr/local/bin/gitlab-ci-multi-runner https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-ci-multi-runner-freebsd-amd64
 sudo wget -O /usr/local/bin/gitlab-ci-multi-runner https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-ci-multi-runner-freebsd-386
```

Give it permissions to execute:

```bash
sudo chmod +x /usr/local/bin/gitlab-ci-multi-runner
```

Create empty log file with correct permissions:

```bash
sudo touch /var/log/gitlab_runner.log && sudo chown gitlab-runner:gitlab-runner /var/log/gitlab_runner.log
```

Create rc.d script:

```bash
cat > /etc/rc.d/gitlab_runner << "EOF"
#!/bin/sh
# PROVIDE: gitlab_runner
# REQUIRE: DAEMON NETWORKING
# BEFORE:
# KEYWORD:

. /etc/rc.subr

name="gitlab_runner"
rcvar="gitlab_runner_enable"

load_rc_config $name

user="gitlab-runner"
user_home="/home/gitlab-runner"
command="/usr/local/bin/gitlab-ci-multi-runner run"
pidfile="/var/run/${name}.pid"

start_cmd="gitlab_runner_start"
stop_cmd="gitlab_runner_stop"
status_cmd="gitlab_runner_status"

gitlab_runner_start()
{
    export USER=${user}
    export HOME=${user_home}
    if checkyesno ${rcvar}; then
        cd ${user_home}
        /usr/sbin/daemon -u ${user} -p ${pidfile} ${command} > /var/log/gitlab_runner.log 2>&1
    fi
}

gitlab_runner_stop()
{
    if [ -f ${pidfile} ]; then
        kill `cat ${pidfile}`
    fi
}

gitlab_runner_status()
{
    if [ ! -f ${pidfile} ] || kill -0 `cat ${pidfile}`; then
        echo "Service ${name} is not running."
    else
        echo "${name} appears to be running."
    fi
}

run_rc_command $1
EOF
```

Make it executable:

```bash
sudo chmod +x /etc/rc.d/gitlab_runner
```


Register the runner:

```bash
sudo -u gitlab-runner -H /usr/local/bin/gitlab-ci-multi-runner register

Please enter the gitlab-ci coordinator URL (e.g. https://gitlab.com):

Please enter the gitlab-ci token for this runner:

Please enter the gitlab-ci description for this runner:
[name]:
Please enter the gitlab-ci tags for this runner (comma separated):

Registering runner... succeeded
Please enter the executor: virtualbox, ssh, shell, parallels, docker, docker-ssh:
shell
Runner registered successfully. Feel free to start it, but if it\'s running already the config should be automatically reloaded!
```

Enable gitlab-runner service and start it:

```bash
sudo sysrc -f /etc/rc.conf "gitlab_runner_enable=YES" 
sudo service gitlab_runner start
```

If you don't want to enable gitlab-runner server to restart after reboot, you can start it:

```bash
sudo service gitlab_runner onestart
```

**The FreeBSD version is also available from [Bleeding edge](bleeding-edge.md)**

Make sure that you read the [FAQ](../faq/README.md) section which describes
some of the most common problems with GitLab Runner.
