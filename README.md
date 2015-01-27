## GitLab CI Multi-purpose Runner

This is GitLab CI Multi-purpose Runner repository an **unofficial GitLab CI runner written in Go**, this application run tests and sends the results to GitLab CI.
[GitLab CI](https://about.gitlab.com/gitlab-ci) is the open-source continuous integration server that coordinates the testing.

This project was made as Go learning opportunity. The initial release was created within two days.

**This is ALPHA. It should work, but also may not.**

[![Build Status](https://travis-ci.org/ayufan/gitlab-ci-multi-runner.svg?branch=master)](https://travis-ci.org/ayufan/gitlab-ci-multi-runner)

### Requirements

**None. This project is designed for the Linux and OS X operating systems.**

### Features

* Allows to run:
 - multiple jobs concurrently
 - use multiple tokens with multiple server (even per-project)
 - limit number of concurrent jobs per-token
* Jobs can be run:
 - locally
 - using Docker container
 - using Docker container and executing job over SSH
 - connecting to remote SSH server
* Is written in Go and distributed as single binary without any other requirements
* Works on Ubuntu, Debian and OS X (should also work on other Linux distributions)
* Allows to customize job running environment
* Automatic configuration reload without restart
* Easy to use setup with support for docker, docker-ssh or ssh running environments
* Enables caching of Docker containers

### Install and initial configuration

1. Simply download one of this binaries for your system:
	```bash
	sudo wget -O /usr/local/bin/gitlab-ci-multi-runner https://github.com/ayufan/gitlab-ci-multi-runner/releases/download/v0.1.0/gitlab-ci-multi-runner-linux-386
	sudo wget -O /usr/local/bin/gitlab-ci-multi-runner https://github.com/ayufan/gitlab-ci-multi-runner/releases/download/v0.1.0/gitlab-ci-multi-runner-linux-amd64
	sudo wget -O /usr/local/bin/gitlab-ci-multi-runner https://github.com/ayufan/gitlab-ci-multi-runner/releases/download/v0.1.0/gitlab-ci-multi-runner-darwin-386
	sudo wget -O /usr/local/bin/gitlab-ci-multi-runner https://github.com/ayufan/gitlab-ci-multi-runner/releases/download/v0.1.0/gitlab-ci-multi-runner-darwin-amd64
	```

1. Give it permissions to execute:
	```bash
	sudo chmod +x /usr/local/bin/gitlab-ci-multi-runner
	```

1. Create a GitLab CI user (Linux)
	```
	sudo adduser --disabled-login --gecos 'GitLab Runner' gitlab_ci_runner
	sudo su gitlab_ci_runner
	cd ~/
	```

1. Setup the runner
	```bash
	$ gitlab-ci-multi-runner-linux setup
	Please enter the gitlab-ci coordinator URL (e.g. http://gitlab-ci.org:3000/ )
	https://ci.gitlab.org/
	Please enter the gitlab-ci token for this runner
	xxx
	Please enter the gitlab-ci hostname for this runner
	my-runner
	Please enter the tag list separated by comma or leave it empty
	linux, lab, worker, ruby
	INFO[0034] fcf5c619 Registering runner... succeeded
	Please enter the executor: shell, docker, docker-ssh, ssh?
	docker
	Please enter the Docker image (eg. ruby:2.1):
	ruby:2.1
	INFO[0037] Runner registered successfully. Feel free to start it, but if it's running already the config should be automatically reloaded!
	```

	* Definition of hostname will be available with version 7.8.0 of GitLab CI. 
	* Ability to specify tag list will be available once this get merged: https://gitlab.com/gitlab-org/gitlab-ci/merge_requests/32

1. Run the runner
	```bash
	$ screen
	$ gitlab-ci-multi-runner run
	```

1. Add to cron
	```bash
	$ crontab -e
	@reboot gitlab-ci-multi-runner run &>log/gitlab-ci-multi-runner.log
	```

### Extra projects?

If you want to add another project, token or image simply re-run setup. *You don't have to re-run the runner. He will automatically reload configuration once it changes.*

### Config file

Configuration uses TOML format described here: https://github.com/toml-lang/toml

1. The global section:
    ```
    concurrent = 4
    root_dir = ""
    ```
    
    This defines global settings of multi-runner:
    * `concurrent` - limits how many jobs globally can be run concurrently. The most upper limit of jobs using all defined runners
    * `root_dir` - allows to change relative dir where all builds, caches, etc. are stored. By default is current working directory

1. The [[runners]] section:
    ```
    [[runners]]
      name = "ruby-2.1-docker"
      url = "https://CI/"
      token = "TOKEN"
      limit = 0
      executor = "docker"
      builds_dir = ""
      shell_script = ""
      environment = ["ENV=value", "LC_ALL=en_US.UTF-8"]
    ```

    This defines one runner entry:
    * `name` - not used, just informatory
    * `url` - CI URL
    * `token` - runner token
    * `limit` - limit how many jobs can be handled concurrently by this token. 0 simply means don't limit.
    * `executor` - select how project should be built. See below.
    * `builds_dir` - directory where builds will be stored in context of selected executor (Locally, Docker, SSH)

1. The EXECUTORS:

    There are a couple of available executors currently:
    * **shell** - run build locally, default
    * **docker** - run build using Docker container - this requires the presence of *[runners.docker]*
    * **docker-ssh** - run build using Docker container, but connect to it with SSH - this requires the presence of *[runners.docker]* and *[runners.ssh]*
    * **ssh** - run build remotely with SSH - this requires the presence of *[runners.ssh]*

1. The [runners.docker] section:
    ```
    [runners.docker]
      host = ""
      image = "ruby:2.1"
      privileged = false
      disable_cache = false
      disable_pull = false
      cache_dir = ""
      registry = ""
      volumes = ["/data", "/home/project/cache"]
      extra_hosts = ["other-host:127.0.0.1"]
      links = ["mysql_container:mysql"]
      services = ["mysql", "redis:2.8", "postgres:9"]
    ```
    
    This defines the Docker Container parameters:
    * `host` - use custom Docker endpoint, by default *DOCKER_HOST* environment is used or *"unix:///var/run/docker.sock"*
    * `image` - use this image to run builds
    * `privileged` - make container run in Privileged mode (insecure)
    * `disable_cache` - disable automatic
    * `disable_pull` - disable automatic image pulling if not found
    * `cache_dir` - specify where Docker caches should be stored (this can be absolute or relative to current working directory)
    * `registry` - specify custom Docker registry to be used
    * `volumes` - specify additional volumes that should be cached
    * `extra_hosts` - specify hosts that should be defined in container environment
    * `links` - specify containers which should be linked with building container
    * `services` - specify additional services that should be run with build. Please visit [Docker Registry](https://registry.hub.docker.com/) for list of available applications. Each service will be run in separate container and linked to the build.

1. The [runners.ssh] section:
    ```
    [runners.ssh]
      host = "my-production-server"
      port = "22"
      user = "root"
      password = "production-server-password"
    ```
    
    This defines the SSH connection parameters:
    * `host` - where to connect (it's override when using *docker-ssh*)
    * `port` - specify port, default: 22
    * `user` - specify user
    * `password` - specify password

1. Example configuration file
    [Example configuration file](config.toml.example)

### FAQ

TBD

### Future

* It should be simple to add additional executors: DigitalOcean? Amazon EC2?
* Tests!

### Author

[Kamil Trzciński](mailto:ayufan@ayufan.eu), 2015, [Polidea](http://www.polidea.com/)

### License

GPLv3
