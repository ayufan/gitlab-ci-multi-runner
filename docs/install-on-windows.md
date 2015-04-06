### Install on Windows

1. Create some folder somewhere in your system, ex.: `C:\Multi-Runner`.

1. Download binary for your system from https://github.com/ayufan/gitlab-ci-multi-runner/releases and put it into previously saved folder.

1. Run `Administrator` command prompt. How to do that is described here: http://pcsupport.about.com/od/windows-8/a/elevated-command-prompt-windows-8.htm. The simplest is to write `Command Prompt` in Windows search field and press `Windows+K`. You will be asked to confirm that you want to execute elevated command prompt.

1. Go to created folder: `cd C:\Multi-Runner`.

1. Setup the runner
	```batch
	$ C:\
	$ cd C:\Multi-Runner
	$ gitlab-ci-multi-runner-linux setup
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
	INFO[0037] Runner registered successfully. Feel free to start it, but if it's running already the config should be automatically reloaded!
	```

1. Install runner as service and start it. You have to enter valid password for current user account, because it's required to start the service by Windows.
	```bash
	$ gitlab-ci-multi-runner install --password ENTER-YOUR-PASSWORD
	$ gitlab-ci-multi-runner start
	```

1. Voila! Runner is installed and will be run after system reboot.

1. Logs are stored in Windows Event Log.

#### Update

1. Stop service (you need elevated command prompt as before):
	```batch
	$ C:\
	$ cd C:\Multi-Runner
	$ gitlab-ci-multi-runner-linux stop
	```

1. Download binary for your system from https://github.com/ayufan/gitlab-ci-multi-runner/releases and replace runner's executable.

1. Start service:
	```batch
	$ gitlab-ci-multi-runner start
	```
