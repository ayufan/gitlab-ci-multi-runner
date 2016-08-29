### Install on OSX

(In the future there will be a brew package).

Download the binary for your system:

```bash
sudo curl --output /usr/local/bin/gitlab-ci-multi-runner https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-ci-multi-runner-darwin-amd64
```

Give it permissions to execute:

```bash
sudo chmod +x /usr/local/bin/gitlab-ci-multi-runner
```

**The rest of commands execute as the user who will run the runner.**

Register the runner:
```bash
gitlab-ci-multi-runner register

Please enter the gitlab-ci coordinator URL (e.g. https://gitlab.com )
https://gitlab.com
Please enter the gitlab-ci token for this runner
xxx
Please enter the gitlab-ci description for this runner
my-runner
INFO[0034] fcf5c619 Registering runner... succeeded
Please enter the executor: shell, docker, docker-ssh, ssh?
docker
Please enter the Docker image (eg. ruby:2.1):
ruby:2.1
INFO[0037] Runner registered successfully. Feel free to start it, but if it's
running already the config should be automatically reloaded!
```

Install runner as service and start it:

```bash
cd ~
gitlab-ci-multi-runner install
gitlab-ci-multi-runner start
```

Voila! Runner is installed and will be run after system reboot.

### Update

Stop the service:

```bash
gitlab-ci-multi-runner stop
```

Download binary to replace runner's executable:

```bash
curl -o /usr/local/bin/gitlab-ci-multi-runner https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-ci-multi-runner-darwin-amd64
```

Give it permissions to execute:

```bash
chmod +x /usr/local/bin/gitlab-ci-multi-runner
```

Start the service:

```bash
gitlab-ci-multi-runner start
```

Make sure that you read the [FAQ](../faq/README.md) section which describes
some of the most common problems with GitLab Runner.

### Limitations on OSX

>**Note:**
The service needs to be installed from the Terminal by running its GUI
interface as your current user. Only then will you be able to manage the service.

Currently, the only proven to work mode for OSX is running service in user-mode.

Since the service will be running only when the user is logged in, you should
enable auto-logging on your OSX machine.

The service will be launched as one of `LaunchAgents`. By using `LaunchAgents`,
the builds will be able to do UI interactions, making it possible to run and
test on the iOS simulator.

It's worth noting that OSX also has `LaunchDaemons`, the services running
completely in background. `LaunchDaemons` are run on system startup, but they
don't have the same access to UI interactions as `LaunchAgents`. You can try to
run the Runner's service as `LaunchDaemon`, but this mode of operation is not
currently supported.

You can verify that the Runner created the service configuration file after
executing the `install` command, by checking the
`~user/Library/LaunchAgents/gitlab-runner.plist` file.

### Upgrade the service file

In order to upgrade the `LaunchAgent` configuration, you need to uninstall and
install the service:

```bash
gitlab-ci-multi-runner uninstall
gitlab-ci-multi-runner install
gitlab-ci-multi-runner start
```
