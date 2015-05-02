### Install on Windows

1. Create a folder somewhere in your system, ex.: `C:\Multi-Runner`.

1. Download the binary for [x86][]  or [amd64][] and put it into the folder you
  created.

1. Run an `Administrator` command prompt ([How to][prompt]). The simplest is to
  write `Command Prompt` in Windows search field, right click and select
  `Run as administrator`. You will be asked to confirm that you want to execute
  the elevated command prompt.

1. Go to the folder you created before: `cd C:\Multi-Runner`.

1. Setup the runner
	```batch
	$ cd C:\Multi-Runner
	$ gitlab-ci-multi-runner-linux setup
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
	INFO[0037] Runner registered successfully. Feel free to start it, but if it's running already the config should be automatically reloaded!
	```

1. Install runner as a service and start it. You have to enter a valid password
  for the current user account, because it's required to start the service by Windows.
	```bash
	$ gitlab-ci-multi-runner install --password ENTER-YOUR-PASSWORD
	$ gitlab-ci-multi-runner start
	```

1. Voila! Runner is installed and will be run after system reboot.

1. Logs are stored in Windows Event Log.

#### Update

1. Stop service (you need elevated command prompt as before):
	```batch
	$ cd C:\Multi-Runner
	$ gitlab-ci-multi-runner stop
	```

1. Download the binary for [x86][] or [amd64][] and replace runner's executable.

1. Start service:
	```batch
	$ gitlab-ci-multi-runner start
	```

[x86]: https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-ci-multi-runner-windows-386.exe
[amd64]: https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-ci-multi-runner-windows-amd64.exe
[prompt]: http://pcsupport.about.com/od/windows-8/a/elevated-command-prompt-windows-8.htm
