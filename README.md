## GitLab Runner

This is the repository of the official GitLab Runner written in Go.
It runs tests and sends the results to GitLab.
[GitLab CI](https://about.gitlab.com/gitlab-ci) is the open-source
continuous integration service included with GitLab that coordinates the testing.

![Build Status](https://gitlab.com/gitlab-org/gitlab-ci-multi-runner/badges/master/build.svg)

### Contributing

The official repository for this project is on [GitLab.com](https://gitlab.com/gitlab-org/gitlab-ci-multi-runner).

* [Development](docs/development/README.md)
* [Issues](https://gitlab.com/gitlab-org/gitlab-ci-multi-runner/issues)
* [Merge Requests](https://gitlab.com/gitlab-org/gitlab-ci-multi-runner/merge_requests)
* [Prepare development environment](docs/development/README.md)

#### Closing issues and merge requests

GitLab is growing very fast and we have a limited resources to deal with reported issues
and merge requests opened by the community volunteers. We appreciate all the contributions
coming from our community. But to help all of us with issues and merge requests management
we need to create some closing policy.

If an issue or merge request has a ~"waiting for feedback" label and the response from the
reporter has not been received for 14 days, we can close it using the following response
template:

```
We haven't received an update for more than 14 days so we will assume that the
problem is fixed or is no longer valid. If you still experience the same problem
try upgrading to the latest version. If the issue persists, reopen this issue
or merge request with the relevant information.
```

### Requirements

**None:** GitLab Runner is run as a single binary.

This project is designed to run on the Linux, OS X, and Windows operating systems.
Other operating systems will probably work as long as you can compile a Go binary on them.

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
 - using Docker container with autoscaling on different clouds and virtualization hypervisors
 - connecting to remote SSH server
* Is written in Go and distributed as single binary without any other requirements
* Supports Bash, Windows Batch and Windows PowerShell
* Works on Ubuntu, Debian, OS X and Windows (and anywhere you can run Docker)
* Allows to customize job running environment
* Automatic configuration reload without restart
* Easy to use setup with support for docker, docker-ssh, parallels or ssh running environments
* Enables caching of Docker containers
* Easy installation as service for Linux, OSX and Windows

### Compatibility chart

Supported features by different executors:

| Executor                              | Shell   | Docker | Docker-SSH | VirtualBox | Parallels | SSH  |
|---------------------------------------|---------|--------|------------|------------|-----------|------|
| Secure Variables                      | ✓       | ✓      | ✓          | ✓          | ✓         | ✓    |
| GitLab Runner Exec command            | ✓       | ✓      | ✓          | no         | no        | no   |
| gitlab-ci.yml: image                  | no      | ✓      | ✓          | no         | no        | no   |
| gitlab-ci.yml: services               | no      | ✓      | ✓          | no         | no        | no   |
| gitlab-ci.yml: cache                  | ✓       | ✓      | no         | no         | no        | no   |
| gitlab-ci.yml: artifacts              | ✓       | ✓      | no         | no         | no        | no   |
| Absolute paths: caching, artifacts    | no      | no     | no         | no         | no        | no   |
| Passing artifacts between stages      | ✓       | ✓      | no         | no         | no        | no   |

Supported systems by different shells:

| Shells                                | Bash        | Windows Batch  | PowerShell |
|---------------------------------------|-------------|----------------|------------|
| Windows                               | ✓           | ✓ (default)    | ✓          |
| Linux                                 | ✓ (default) | no             | no         |
| OSX                                   | ✓ (default) | no             | no         |
| FreeBSD                               | ✓ (default) | no             | no         |

### Install GitLab Runner

* [Install using GitLab's repository for Debian/Ubuntu/CentOS/RedHat (preferred)](docs/install/linux-repository.md)
* [Install on OSX (preferred)](docs/install/osx.md)
* [Install on Windows (preferred)](docs/install/windows.md)
* [Install as Docker Service](docs/install/docker.md)
* [Install in Auto-scaling mode](docs/install/autoscaling.md)
* [Use on FreeBSD](docs/install/freebsd.md)

### Use GitLab Runner

* [See the **commands** documentation](docs/commands/README.md)
* [Use self-signed certificates](docs/configuration/tls-self-signed.md)
* [Cleanup the docker images automatically](https://gitlab.com/gitlab-org/gitlab-runner-docker-cleanup)

### Select executor

* [Help me select executor](docs/executors/README.md#imnotsure)
* [Shell](docs/executors/shell.md)
* [Docker and Docker-SSH](docs/executors/docker.md)
* [Parallels](docs/executors/parallels.md)
* [VirtualBox](docs/executors/virtualbox.md)
* [SSH](docs/executors/ssh.md)

### Troubleshooting

* [FAQ](docs/faq/README.md)

### Advanced Configuration

* [Auto-scaling](docs/configuration/autoscale.md)
* [Install Bleeding Edge (development)](docs/install/bleeding-edge.md)
* [Manual installation (advanced)](docs/install/linux-manually.md)
* [See details about the shells](docs/shells/README.md)
* [See advanced configuration options](docs/configuration/advanced-configuration.md)
* [See security considerations](docs/security/index.md)

### Extra projects?

If you want to add another project, token or image simply RE-RUN SETUP.
*You don't have to re-run the runner. It will automatically reload configuration once it changes.*

### Changelog

Visit [Changelog](CHANGELOG.md) to view recent changes.

#### Version 0.5.0

Version 0.5.0 introduces many security related changes.
One of such changes is the different location of `config.toml`.
Previously (prior 0.5.0) config was read from current working directory.
Currently, when `gitlab-runner` is executed by `root` or with `sudo` config is read from `/etc/gitlab-runner/config.toml`.
If `gitlab-runner` is executed by non-root user, the config is read from `$HOME/.gitlab-runner/config.toml`.
However, this doesn't apply to Windows where config is still read from current working directory, but this most likely will change in future.

The config file is automatically migrated when GitLab Runner was installed from GitLab's repository.
**For manual installations the config needs to be moved by hand.**

### The future

* Please see the [GitLab Direction page](https://about.gitlab.com/direction/).
* Feel free submit issues with feature proposals on the issue tracker.

### Author

[Kamil Trzciński](mailto:ayufan@ayufan.eu)

### License

This code is distributed under the MIT license, see the LICENSE file.
