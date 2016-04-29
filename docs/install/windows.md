### Install on Windows

Create a folder somewhere in your system, ex.: `C:\Multi-Runner`.

Download the binary for [x86][]  or [amd64][] and put it into the folder you
created.

Run an `Administrator` command prompt ([How to][prompt]). The simplest is to
write `Command Prompt` in Windows search field, right click and select
`Run as administrator`. You will be asked to confirm that you want to execute
the elevated command prompt.

Register the runner:

```bash
cd C:\Multi-Runner
gitlab-ci-multi-runner register

Please enter the gitlab-ci coordinator URL (e.g. http://gitlab-ci.org:3000/ )
https://ci.gitlab.com
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

Install runner as a service and start it. You have to enter a valid password
for the current user account, because it's required to start the service by Windows:

```bash
gitlab-ci-multi-runner install --user ENTER-YOUR-USERNAME --password ENTER-YOUR-PASSWORD
gitlab-ci-multi-runner start
```

> **Notice:** If you don't have a Windows Password, Runner's service won't start. To
> fix this please read [How to Configure the Service to Start Up with the Built-in System Account](https://support.microsoft.com/en-us/kb/327545#bookmark-6)
> on Microsoft's support website.

Voila! Runner is installed and will be run after system reboot.

Logs are stored in Windows Event Log.

#### Update

Stop service (you need elevated command prompt as before):

```bash
cd C:\Multi-Runner
gitlab-ci-multi-runner stop
```

Download the binary for [x86][] or [amd64][] and replace runner's executable.

Start service:

```bash
gitlab-ci-multi-runner start
```

Make sure that you read the [FAQ](../faq/README.md) section which describes
some of the most common problems with GitLab Runner.

[x86]: https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-ci-multi-runner-windows-386.exe
[amd64]: https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-ci-multi-runner-windows-amd64.exe
[prompt]: http://pcsupport.about.com/od/windows-8/a/elevated-command-prompt-windows-8.htm
