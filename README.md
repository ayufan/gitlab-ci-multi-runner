## GitLab Runner

This is the repository of the official GitLab Runner written in Go.
It runs tests and sends the results to GitLab.
[GitLab CI](https://about.gitlab.com/gitlab-ci) is the open-source
continuous integration service included with GitLab that coordinates the testing.

[![Build Status](https://ci.gitlab.com/projects/1885/status.png?ref=master)](https://ci.gitlab.com/projects/1885?ref=master)

### Contributing

The official repository for this project is on [GitLab.com](https://gitlab.com/gitlab-org/gitlab-ci-multi-runner).

* [Development](docs/development/README.md)
* [Issues](https://gitlab.com/gitlab-org/gitlab-ci-multi-runner/issues)
* [Merge Requests](https://gitlab.com/gitlab-org/gitlab-ci-multi-runner/merge_requests)

### Requirements

**None:** GitLab Runner is run as a single binary.

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

### Version 0.5.0

Version 0.5.0 introduces many security related changes.
One of such changes is the different location of `config.toml`.
Previously (prior 0.5.0) config was read from current working directory.
Currently, when `gitlab-runner` is executed by `root` or with `sudo` config is read from `/etc/gitlab-runner/config.toml`.
If `gitlab-runner` is executed by non-root user, the config is read from `$HOME/.gitlab-runner/config.toml`.
However, this doesn't apply to Windows where config is still read from current working directory, but this most likely will change in future.

The config file is automatically migrated when GitLab Runner was installed from GitLab's repository.
**For manual installations the config needs to be moved by hand.**

### Installation

* [Install using GitLab's repository for Debian/Ubuntu/CentOS/RedHat (preferred)](docs/install/linux-repository.md)
* [Install on OSX (preferred)](docs/install/osx.md)
* [Install on Windows (preferred)](docs/install/windows.md)
* [Install as Docker Service](docs/install/docker.md)
* [Use on FreeBSD](docs/install/freebsd.md)
* [Manual installation (advanced)](docs/install/linux-manually.md)
* [Bleeding edge (development)](docs/install/bleeding-edge.md)
* [Install development environment](docs/development/README.md)

### Troubleshoting

* [FAQ](docs/faq/README.md)

### Advanced Configuration

* [See the self-signed certificates](docs/configuration/tls-self-signed.md)
* [See advanced configuration options](docs/configuration/advanced-configuration.md)
* [See example configuration file](config.toml.example)
* [See security considerations](docs/security/index.md)
* [Example configuration running the GitLab CE integration tests](docs/examples/gitlab.md)

### Cleaning docker images automatically

* [GitLab Runner Docker Cleanup tool](https://gitlab.com/gitlab-org/gitlab-runner-docker-cleanup)

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

* Please see the [GitLab Direction page](https://about.gitlab.com/direction/).
* Feel free submit issues with feature proposals on the issue tracker.

### Author

[Kamil Trzciński](mailto:ayufan@ayufan.eu)

### License

GPLv3