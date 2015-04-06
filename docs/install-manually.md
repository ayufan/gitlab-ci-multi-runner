### Manual installation and configuration (for other distributions)

1. Simply download one of this binaries for your system (modify v0.1.13 with latest release number):
	```bash
	sudo wget -O /usr/local/bin/gitlab-ci-multi-runner https://github.com/ayufan/gitlab-ci-multi-runner/releases/download/v0.1.13/gitlab-ci-multi-runner-linux-386
	sudo wget -O /usr/local/bin/gitlab-ci-multi-runner https://github.com/ayufan/gitlab-ci-multi-runner/releases/download/v0.1.13/gitlab-ci-multi-runner-linux-amd64
	```

1. Give it permissions to execute:
	```bash
	sudo chmod +x /usr/local/bin/gitlab-ci-multi-runner
	```

1. If you want to use Docker - install Docker:
    ```bash
    curl -sSL https://get.docker.com/ | sh
    ```

1. Create a GitLab CI user (on Linux)
	```
	sudo useradd --comment 'GitLab Runner' --create-home gitlab_ci_multi_runner --shell /bin/bash
	sudo usermod -aG docker gitlab_ci_multi_runner
	```

1. Setup the runner
	```bash
	cd ~gitlab_ci_multi_runner
	sudo -u gitlab_ci_multi_runner -H gitlab-ci-multi-runner-linux setup
	```

1. Secure config.toml
	```bash
    sudo chmod 0600 config.toml
    ```

1. Install and run as service
	```bash
	sudo gitlab-ci-multi-runner-linux install --user=gitlab_ci_multi_runner
	sudo gitlab-ci-multi-runner-linux start
	```

#### Update

1. Stop service (you need elevated command prompt as before):
	```bash
	sudo gitlab-ci-multi-runner-linux stop
	```

1. Download binary for your system from https://github.com/ayufan/gitlab-ci-multi-runner/releases and replace runner's executable:
	```bash
	sudo wget -O /usr/local/bin/gitlab-ci-multi-runner https://github.com/ayufan/gitlab-ci-multi-runner/releases/download/v0.1.13/gitlab-ci-multi-runner-linux-386
	sudo wget -O /usr/local/bin/gitlab-ci-multi-runner https://github.com/ayufan/gitlab-ci-multi-runner/releases/download/v0.1.13/gitlab-ci-multi-runner-linux-amd64
	```

1. Give it permissions to execute:
	```bash
	sudo chmod +x /usr/local/bin/gitlab-ci-multi-runner
	```

1. Start service:
	```bash
	sudo gitlab-ci-multi-runner start
	```
