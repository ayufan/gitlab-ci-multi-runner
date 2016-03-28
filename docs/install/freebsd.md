### Install on FreeBSD

Download the binary for your system:

```bash
sudo wget -O /usr/local/bin/gitlab-ci-multi-runner https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-ci-multi-runner-freebsd-amd64
sudo wget -O /usr/local/bin/gitlab-ci-multi-runner https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-ci-multi-runner-freebsd-386
```

Give it permissions to execute:

```bash
sudo chmod +x /usr/local/bin/gitlab-ci-multi-runner
```

**The rest of commands execute as the user who will run the runner.**

Register the runner:
```bash
gitlab-ci-multi-runner register

Please enter the gitlab-ci coordinator URL (e.g. https://gitlab.com/ci):

Please enter the gitlab-ci token for this runner:

Please enter the gitlab-ci description for this runner:
[name]:
Please enter the gitlab-ci tags for this runner (comma separated):

Registering runner... succeeded
Please enter the executor: virtualbox, ssh, shell, parallels, docker, docker-ssh:
shell
Runner registered successfully. Feel free to start it, but if it's running already the config should be automatically reloaded!
```

Run GitLab-Runner:

```bash
cd ~
gitlab-ci-multi-runner run
```

Voila! Runner is currently running, but it will not start automatically after system reboot because BSD startup service is not supported.

**The FreeBSD version is also available from [Bleeding edge](bleeding-edge.md)**

Make sure that you read the [FAQ](../faq/README.md) section which describes
some of the most common problems with GitLab Runner.
