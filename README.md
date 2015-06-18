## GitLab Runner

This is the GitLab Runner repository, the official GitLab CI
runner written in Go. It runs tests and sends the results to GitLab CI.
[GitLab CI](https://about.gitlab.com/gitlab-ci) is the open-source
continuous integration server that coordinates the testing.

[![Build Status](https://ci.gitlab.com/projects/1885/status.png?ref=master)](https://ci.gitlab.com/projects/1885?ref=master)

### Contributing

The official repository for this project is on [GitLab.com](https://gitlab.com/gitlab-org/gitlab-ci-multi-runner).

* [Issues](https://gitlab.com/gitlab-org/gitlab-ci-multi-runner/issues)
* [Merge Requests](https://gitlab.com/gitlab-org/gitlab-ci-multi-runner/merge_requests)

### Requirements

**None:** gitlab-ci-multi-runner is run as a single binary.

This project is designed for the Linux, OS X and Windows operating systems.

If you want to use **Docker** make sure that you have **1.5.0** at least installed.

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

### Installation

* [Install using GitLab's repository for Debian/Ubuntu/CentOS/RedHat (preferred)](docs/install/linux-repository.md)
* [Install on OSX (preffered)](docs/install/osx.md)
* [Install on Windows (preffered)](docs/install/windows.md)
* [Install as Docker Service](docs/install/docker.md)
* [Manual installation (advanced)](docs/install/linux-manually.md)
* [Bleeding edge (development)](docs/install/bleeding-edge.md)

### Advanced Configuration

* [See advanced configuration options](docs/configuration/advanced-configuration.md)
* [See example configuration file](config.toml.example)
* [See security considerations](docs/security/index.md)

### Example integrations

* [Integrate GitLab CE](docs/examples/gitlab.md)
* [Integrate GitLab CI](docs/examples/gitlab-ci.md)

### Extra projects?

If you want to add another project, token or image simply RE-RUN SETUP.
*You don't have to re-run the runner. It will automatically reload configuration once it changes.*

### Changelog

Visit [Changelog](CHANGELOG.md) to view recent changes.

### Help

```bash
$ gitlab-ci-multi-runner --help
NAME:
   gitlab-ci-multi-runner - a GitLab Runner

USAGE:
   gitlab-ci-multi-runner [global options] command [command options] [arguments...]

VERSION:
   dev

AUTHOR:
  Kamil Trzciński - <ayufan@ayufan.eu>

COMMANDS:
   delete	delete specific runner
   run, r	run multi runner service
   install	install service
   uninstall	uninstall service
   start	start service
   stop		stop service
   restart	restart service
   setup, s	setup a new runner
   run-single	start single runner
   verify	verify all registered runners
   help, h	Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug			debug mode [$DEBUG]
   --log-level, -l 'info'	Log level (options: debug, info, warn, error, fatal, panic)
   --help, -h			show help
   --version, -v		print the version
```

### Future

* It should be simple to add additional executors: DigitalOcean? Amazon EC2?
* Maybe script annotations?

### Author

[Kamil Trzciński](mailto:ayufan@ayufan.eu), 2015, [Polidea](http://www.polidea.com/)

### License

GPLv3
