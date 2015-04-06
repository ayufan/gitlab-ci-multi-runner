### Install on OSX

(In the future there will be brew package)

1. Download binary for your system (modify v0.1.13 with latest release number):
	```bash
	sudo wget -O /usr/local/bin/gitlab-ci-multi-runner https://github.com/ayufan/gitlab-ci-multi-runner/releases/download/v0.1.13/gitlab-ci-multi-runner-darwin-amd64
	```

1. Give it permissions to execute:
	```bash
	sudo chmod +x /usr/local/bin/gitlab-ci-multi-runner
	```

1. The rest of commands execute as user who will run the runner

1. Setup the runner
	```bash
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

1. Install runner as service and start it
	```bash
	$ gitlab-ci-multi-runner install
	$ gitlab-ci-multi-runner start
	```

1. Voila! Runner is installed and will be run after system reboot.

#### Update

1. Stop service (you need elevated command prompt as before):
	```bash
	gitlab-ci-multi-runner-linux stop
	```

1. Download binary for your system from https://github.com/ayufan/gitlab-ci-multi-runner/releases and replace runner's executable:
	```bash
	wget -O /usr/local/bin/gitlab-ci-multi-runner https://github.com/ayufan/gitlab-ci-multi-runner/releases/download/v0.1.13/gitlab-ci-multi-runner-darwin-amd64
	```

1. Give it permissions to execute:
	```bash
	chmod +x /usr/local/bin/gitlab-ci-multi-runner
	```

1. Start service:
	```bash
	gitlab-ci-multi-runner start
	```
