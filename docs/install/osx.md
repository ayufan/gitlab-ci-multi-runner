### Install on OSX

(In the future there will be a brew package).

Download the binary for your system:

```bash
sudo wget -O /usr/local/bin/gitlab-ci-multi-runner https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-ci-multi-runner-darwin-amd64
```

Give it permissions to execute:

```bash
sudo chmod +x /usr/local/bin/gitlab-ci-multi-runner
```

**The rest of commands execute as the user who will run the runner.**

Register the runner:
```bash
gitlab-ci-multi-runner register

Please enter the gitlab-ci coordinator URL (e.g. http://gitlab-ci.org:3000/ )
https://ci.gitlab.org/
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
wget -O /usr/local/bin/gitlab-ci-multi-runner https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-ci-multi-runner-darwin-amd64
```

Give it permissions to execute:

```bash
chmod +x /usr/local/bin/gitlab-ci-multi-runner
```

Start the service:

```bash
gitlab-ci-multi-runner start
```