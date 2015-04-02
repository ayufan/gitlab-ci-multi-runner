## GitLab CI Multi-purpose Runner

This is GitLab CI Multi-purpose Runner repository an **unofficial GitLab CI runner written in Go**, this application run tests and sends the results to GitLab CI.
[GitLab CI](https://about.gitlab.com/gitlab-ci) is the open-source continuous integration server that coordinates the testing.

This project was made as Go learning opportunity. The initial release was created within two days.

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
* Supports Bash, Windows Batch and Windows PowerShell
* Works on Ubuntu, Debian, OS X and Windows (and anywhere you can run Docker)
* Allows to customize job running environment
* Automatic configuration reload without restart
* Easy to use setup with support for docker, docker-ssh, parallels or ssh running environments
* Enables caching of Docker containers
* Easy installation as service for Linux, OSX and Windows

### Install and initial configuration (For Debian, Ubuntu and CentOS)

1. If you want to use Docker runnner install it before:
  ```bash
  curl -sSL https://get.docker.com/ | sh
  ```

1. Add package to apt-get or yum (**THESE REPOSITORIES ARE TEMPORARY AND ARE SUBJECT TO CHANGE**)
  ```bash
  # For Debian/Ubuntu
  curl https://packagecloud.io/install/repositories/ayufan/gitlab-ci-multi-runner/script.deb | sudo bash

  # For CentOS
  curl https://packagecloud.io/install/repositories/ayufan/gitlab-ci-multi-runner/script.rpm | sudo bash
  ```

1. Install `gitlab-ci-multi-runner`
  ```bash
  # For Debian/Ubuntu
  apt-get install gitlab-ci-multi-runner

  # For CentOS
  yum install gitlab-ci-multi-runner
  ```

1. Setup the runner
  ```bash
  $ cd ~gitlab_ci_multi_runner
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

1. Runner should be started already and you are ready to build your projects!

#### Update

1. Simply execute to install latest version
  ```bash
  # For Debian/Ubuntu  
  apt-get update
  apt-get install gitlab-ci-multi-runner

  # For CentOS
  yum install gitlab-ci-multi-runner
  ```

### Manual installation and configuration (For other systems)

1. Simply download one of this binaries for your system:
	```bash
	sudo wget -O /usr/local/bin/gitlab-ci-multi-runner https://github.com/ayufan/gitlab-ci-multi-runner/releases/download/v0.1.13/gitlab-ci-multi-runner-linux-386
	sudo wget -O /usr/local/bin/gitlab-ci-multi-runner https://github.com/ayufan/gitlab-ci-multi-runner/releases/download/v0.1.13/gitlab-ci-multi-runner-linux-amd64
	sudo wget -O /usr/local/bin/gitlab-ci-multi-runner https://github.com/ayufan/gitlab-ci-multi-runner/releases/download/v0.1.13/gitlab-ci-multi-runner-darwin-386
	sudo wget -O /usr/local/bin/gitlab-ci-multi-runner https://github.com/ayufan/gitlab-ci-multi-runner/releases/download/v0.1.13/gitlab-ci-multi-runner-darwin-amd64
	```

1. Give it permissions to execute:
	```bash
	sudo chmod +x /usr/local/bin/gitlab-ci-multi-runner
	```

1. If you want to use Docker - install Docker:
    ```bash
    curl -sSL https://get.docker.com/ | sh
    ```

1. Create a GitLab CI user (Linux)
	```
	sudo useradd --comment 'GitLab Runner' --create-home gitlab_ci_multi_runner
	sudo usermod -aG docker gitlab_ci_multi_runner
	sudo su gitlab_ci_multi_runner
	cd
	```

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

	* Definition of hostname will be available with version 7.8.0 of GitLab CI.

1. Run the runner
	```bash
	$ screen
	$ gitlab-ci-multi-runner run
	```

1. Add to cron
	```bash
	$ crontab -e
	@reboot gitlab-ci-multi-runner run &>gitlab-ci-multi-runner.log
	```

### Docker image installation and configuration (run gitlab-ci-multi-runner in a container)

1. Pull the image (optional):
  ```bash
  $ docker pull ayufan/gitlab-ci-multi-runner:latest
  ```

1. Start the container:

  We need to mount a data volume into our gitlab-ci-multi-runner container to be used for configs and other resources:
  ```bash
  $ docker run -d --name multi-runner --restart always \
      -v /PATH/TO/DATA/FOLDER:/data \
      ayufan/gitlab-ci-multi-runner:latest
  ```

  OR you can use a data container to mount you custom data volume:
  ```bash
  $ docker run -d --name multi-runner-data -v /data \
      busybox:latest /bin/true

  $ docker run -d --name multi-runner --restart always \
      --volumes-from multi-runner-data \
      ayufan/gitlab-ci-multi-runner:latest
  ```

  If you are planning on using Docker as the method of spawing runners you'll need to mount your docker socket like so:
  ```bash
  $ docker run -d --name multi-runner --restart always \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -v /PATH/TO/DATA/FOLDER:/data \
      ayufan/gitlab-ci-multi-runner:latest
  ```

1. Setup the runner:
  ```bash
  $ docker exec -it multi-runner gitlab-ci-multi-runner setup
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

1. Runner should be started already and you are ready to build your projects!

#### Update

1. Pull the latest version:
  ```bash
  $ docker pull ayufan/gitlab-ci-multi-runner:latest
  ```

1. Stop and remove the existing container:
  ```bash
  $ docker stop multi-runner && docker rm multi-runner
  ```

1. Start the container as you did originally:
  ```bash
  $ docker run -d --name multi-runner --restart always \
      --volumes-from multi-runner-data \
      -v /var/run/docker.sock:/var/run/docker.sock \
      ayufan/gitlab-ci-multi-runner:latest
  ```
  **note**: you need to use the same method for mounting you data volume as you did originally (`-v /PATH/TO/DATA/FOLDER:/data` or `--volumes-from multi-runner-data`)

#### Installing Trusted SSL Server Certificates

If your GitLab CI server is using self-signed SSL certificates then you should make sure the GitLab CI server certificate is trusted by the gitlab-ci-multi-runner container for them to be able to talk to each other.

The gitlab-ci-multi-runner image is configured to look for the trusted SSL certificates at `/data/certs/ca.crt`, this can however be changed using the `-e "CA_CERTIFICATES_PATH=/DIR/CERT"` configuration option.

Copy the `ca.crt` file into the `certs` directory on the data volume (or container). The `ca.crt` file should contain the root certificates of all the servers you want gitlab-ci-multi-runner to trust.
The gitlab-ci-multi-runner container will import the `ca.crt` file on startup so if your container is already running you may need to restart it for the changes to take place.

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
      shell = ""
      clean_environment = false
      environment = ["ENV=value", "LC_ALL=en_US.UTF-8"]
      disable_verbose = false
    ```

    This defines one runner entry:
    * `name` - not used, just informatory
    * `url` - CI URL
    * `token` - runner token
    * `limit` - limit how many jobs can be handled concurrently by this token. 0 simply means don't limit.
    * `executor` - select how project should be built. See below.
    * `builds_dir` - directory where builds will be stored in context of selected executor (Locally, Docker, SSH)
    * `clean_environment` - do not inherit any environment variables from the multi-runner process
    * `environment` - append or overwrite environment variables
    * `shell` - the name of shell to generate the script (default value is platform dependent)

1. The EXECUTORS:

    There are a couple of available executors currently:
    * **shell** - run build locally, default
    * **docker** - run build using Docker container - this requires the presence of *[runners.docker]*
    * **docker-ssh** - run build using Docker container, but connect to it with SSH - this requires the presence of *[runners.docker]* and *[runners.ssh]*
    * **ssh** - run build remotely with SSH - this requires the presence of *[runners.ssh]*
    * **parallels** - run build using Parallels VM, but connect to it with SSH - this requires the presence of *[runners.parallels]* and *[runners.ssh]*

1. The SHELLS:

    There are a couple of available shells that can be run on different platforms:
    * **bash** - generate Bash (Bourne-shell) script. All commands executed in Bash context (default for all Unix systems)
    * **cmd** - generate Windows Batch script. All commands are executed in Batch context (default for Windows)
    * **powershell** - generate Windows PowerShell script. All commands are executed in PowerShell context

1. The [runners.docker] section:
    ```
    [runners.docker]
      host = ""
      hostname = ""
      image = "ruby:2.1"
      privileged = false
      disable_cache = false
      disable_pull = false
      wait_for_services_timeout = 30
      cache_dir = ""
      registry = ""
      volumes = ["/data", "/home/project/cache"]
      extra_hosts = ["other-host:127.0.0.1"]
      links = ["mysql_container:mysql"]
      services = ["mysql", "redis:2.8", "postgres:9"]
    ```
    
    This defines the Docker Container parameters:
    * `host` - specify custom Docker endpoint, by default *DOCKER_HOST* environment is used or *"unix:///var/run/docker.sock"*
    * `hostname` - specify custom hostname for Docker container
    * `image` - use this image to run builds
    * `privileged` - make container run in Privileged mode (insecure)
    * `disable_cache` - disable automatic
    * `disable_pull` - disable automatic image pulling if not found
    * `wait_for_services_timeout` - specify how long to wait for docker services, set to 0 to disable, default: 30
    * `cache_dir` - specify where Docker caches should be stored (this can be absolute or relative to current working directory)
    * `registry` - specify custom Docker registry to be used
    * `volumes` - specify additional volumes that should be cached
    * `extra_hosts` - specify hosts that should be defined in container environment
    * `links` - specify containers which should be linked with building container
    * `services` - specify additional services that should be run with build. Please visit [Docker Registry](https://registry.hub.docker.com/) for list of available applications. Each service will be run in separate container and linked to the build.

1. The [runners.parallels] section:
    ```
    [runners.parallels]
      base_name = "my-parallels-image"
      template_name = ""
      disable_snapshots = false
    ```

    This defines the Parallels parameters:
    * `base_name` - name of Parallels VM which will be cloned
    * `template_name` - custom name of Parallels VM linked template (optional)
    * `disable_snapshots` - if disabled the VMs will be destroyed after build

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

## Example integrations

### How to configure runner for GitLab CE integration tests? (uses confined Docker executor)

1. Run setup
    ```bash
    $ gitlab-ci-multi-runner setup \
      --non-interactive \
      --url "https://ci.gitlab.com/" \
      --registration-token "REGISTRATION_TOKEN" \
      --description "gitlab-ce-ruby-2.1" \
      --executor "docker" \
      --docker-image ruby:2.1 --docker-mysql latest \
      --docker-postgres latest --docker-redis latest
    ```

1. Add job to test with MySQL
    ```bash
    wget -q http://ftp.de.debian.org/debian/pool/main/p/phantomjs/phantomjs_1.9.0-1+b1_amd64.deb
    dpkg -i phantomjs_1.9.0-1+b1_amd64.deb

    apt-get update -qq
    apt-get install -y -qq libicu-dev libkrb5-dev cmake nodejs

    bundle install --deployment --path /cache

    cp config/application.yml.example config/application.yml

    cp config/database.yml.mysql config/database.yml
    sed -i 's/username:.*/username: root/g' config/database.yml
    sed -i 's/password:.*/password:/g' config/database.yml
    sed -i 's/# socket:.*/host: mysql/g' config/database.yml

    cp config/resque.yml.example config/resque.yml
    sed -i 's/localhost/redis/g' config/resque.yml

    bundle exec rake db:create

    bundle exec rake test_ci
    ```

1. Add job to test with PostgreSQL
    ```bash
    wget -q http://ftp.de.debian.org/debian/pool/main/p/phantomjs/phantomjs_1.9.0-1+b1_amd64.deb
    dpkg -i phantomjs_1.9.0-1+b1_amd64.deb

    apt-get update -qq
    apt-get install -y -qq libicu-dev libkrb5-dev cmake nodejs

    bundle install --deployment --path /cache

    cp config/application.yml.example config/application.yml

    cp config/database.yml.postgresql config/database.yml
    sed -i 's/username:.*/username: postgres/g' config/database.yml
    sed -i 's/password:.*/password:/g' config/database.yml
    sed -i 's/pool:.*/&\n  host: postgres/g' config/database.yml

    cp config/resque.yml.example config/resque.yml
    sed -i 's/localhost/redis/g' config/resque.yml

    bundle exec rake db:create

    bundle exec rake test_ci
    ```

1. Voila! You now have GitLab CE integration testing instance with bundle caching. Push some commits to test it.

1. Look into `config.toml` and tune it.

### How to configure runner for GitLab CI integration tests? (uses confined Docker executor)

1. Run setup
    ```bash
    $ gitlab-ci-multi-runner setup \
      --non-interactive \
      --url "https://ci.gitlab.com/" \
      --registration-token "REGISTRATION_TOKEN" \
      --description "gitlab-ci-ruby-2.1" \
      --executor "docker" \
      --docker-image ruby:2.1 --docker-mysql latest \
      --docker-postgres latest --docker-redis latest
    ```

1. Add job to test with MySQL
    ```bash
    wget -q http://ftp.de.debian.org/debian/pool/main/p/phantomjs/phantomjs_1.9.0-1+b1_amd64.deb
    dpkg -i phantomjs_1.9.0-1+b1_amd64.deb

    apt-get update -qq
    apt-get install -qq nodejs

    bundle install --deployment --path /cache

    cp config/application.yml.example config/application.yml

    cp config/database.yml.mysql config/database.yml
    sed -i 's/username:.*/username: root/g' config/database.yml
    sed -i 's/password:.*/password:/g' config/database.yml
    sed -i 's/# socket:.*/host: mysql/g' config/database.yml

    cp config/resque.yml.example config/resque.yml
    sed -i 's/localhost/redis/g' config/resque.yml

    bundle exec rake db:create
    bundle exec rake db:setup
    bundle exec rake spec
    ```

1. Add job to test with PostgreSQL
    ```bash
    wget -q http://ftp.de.debian.org/debian/pool/main/p/phantomjs/phantomjs_1.9.0-1+b1_amd64.deb
    dpkg -i phantomjs_1.9.0-1+b1_amd64.deb

    apt-get update -qq
    apt-get install -qq nodejs

    bundle install --deployment --path /cache

    cp config/application.yml.example config/application.yml

    cp config/database.yml.postgresql config/database.yml
    sed -i 's/username:.*/username: postgres/g' config/database.yml
    sed -i 's/password:.*/password:/g' config/database.yml
    sed -i 's/# socket:.*/host: postgres/g' config/database.yml

    cp config/resque.yml.example config/resque.yml
    sed -i 's/localhost/redis/g' config/resque.yml

    bundle exec rake db:create
    bundle exec rake db:setup
    bundle exec rake spec
    ```

1. Voila! You now have GitLab CI integration testing instance with bundle caching. Push some commits to test it.

1. Look into `config.toml` and tune it.

### Changelog

Visit [Changelog](CHANGELOG.md) to view recent changes.

### FAQ

1. Check help `gitlab-ci-multi-runner`:
    ```bash
    gitlab-ci-multi-runner setup --help
    gitlab-ci-multi-runner run --help
    gitlab-ci-multi-runner run-single --help
    ```

### Future

* It should be simple to add additional executors: DigitalOcean? Amazon EC2?
* Tests!

### Author

[Kamil Trzciński](mailto:ayufan@ayufan.eu), 2015, [Polidea](http://www.polidea.com/)

### License

GPLv3
