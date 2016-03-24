# GitLab Runner Commands

GitLab Runner contains a set of commands with which you register, manage and
run your builds.

You can check a recent list of commands by executing:

```bash
gitlab-runner --help
```

Append `--help` after a command to see its specific help page:

```bash
gitlab-runner <command> --help
```

---

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Using environment variables](#using-environment-variables)
- [Running in debug mode](#running-in-debug-mode)
- [Super-user permission](#super-user-permission)
- [Configuration file](#configuration-file)
- [Signals](#signals)
- [Commands overview](#commands-overview)
- [Registration-related commands](#registration-related-commands)
    - [gitlab-runner register](#gitlab-runner-register)
        - [Interactive registration](#interactive-registration)
        - [Non-interactive registration](#non-interactive-registration)
    - [gitlab-runner list](#gitlab-runner-list)
    - [gitlab-runner verify](#gitlab-runner-verify)
    - [gitlab-runner unregister](#gitlab-runner-unregister)
- [Service-related commands](#service-related-commands)
    - [gitlab-runner install](#gitlab-runner-install)
    - [gitlab-runner uninstall](#gitlab-runner-uninstall)
    - [gitlab-runner start](#gitlab-runner-start)
    - [gitlab-runner stop](#gitlab-runner-stop)
    - [gitlab-runner restart](#gitlab-runner-restart)
    - [gitlab-runner status](#gitlab-runner-status)
    - [Multiple services](#multiple-services)
- [Run-related commands](#run-related-commands)
    - [gitlab-runner run](#gitlab-runner-run)
    - [gitlab-runner run-single](#gitlab-runner-run-single)
    - [gitlab-runner exec](#gitlab-runner-exec)
    - [Limitations of `gitlab-runner exec`](#limitations-of-gitlab-runner-exec)
- [Internal commands](#internal-commands)
    - [gitlab-runner artifacts-downloader](#gitlab-runner-artifacts-downloader)
    - [gitlab-runner artifacts-uploader](#gitlab-runner-artifacts-uploader)
    - [gitlab-runner cache-archiver](#gitlab-runner-cache-archiver)
    - [gitlab-runner cache-extractor](#gitlab-runner-cache-extractor)
- [Troubleshooting](#troubleshooting)
    - [**Access Denied** when running the service-related commands](#access-denied-when-running-the-service-related-commands)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Using environment variables

Most of the commands support environment variables as a method to pass the
configuration to the command.

You can see the name of the environment variable when invoking `--help` for a
specific command. For example, you can see below the help message for the `run`
command:

```bash
gitlab-runner run --help
```

The output would be similar to:

```bash
NAME:
   gitlab-runner run - run multi runner service

USAGE:
   gitlab-runner run [command options] [arguments...]

OPTIONS:
   -c, --config "/Users/ayufan/.gitlab-runner/config.toml"	Config file [$CONFIG_FILE]
```

## Running in debug mode

Debug mode is especially useful when looking for the cause of some undefined
behavior or error.

To run a command in debug mode, prepend the command with `--debug`:

```bash
gitlab-runner --debug <command>
```

## Super-user permission

Commands that access the configuration of GitLab Runner behave differently when
executed as super-user (`root`). The file location depends on the user executing
the command.

Be aware of the notice that is written when executing the commands that are
used for running builds, registering services or managing registered runners:

```bash
gitlab-runner run

INFO[0000] Starting multi-runner from /Users/ayufan/.gitlab-runner/config.toml ...  builds=0
WARN[0000] Running in user-mode.
WARN[0000] Use sudo for system-mode:
WARN[0000] $ sudo gitlab-runner...
```

You should use `user-mode` if you are really sure that this is a mode that you
want to work with. Otherwise, prefix your command with `sudo`:

```
sudo gitlab-runner run

INFO[0000] Starting multi-runner from /etc/gitlab-runner/config.toml ...  builds=0
INFO[0000] Running in system-mode.
```

In the case of **Windows** you may need to run the **Command Prompt** in
**Administrative Mode**.

## Configuration file

GitLab Runner configuration uses the [TOML] format.

The file to be edited can be found in:

1. `/etc/gitlab-runner/config.toml` on \*nix systems when gitlab-runner is
   executed as super-user (`root`)
1. `~/.gitlab-runner/config.toml` on \*nix systems when gitlab-runner is
   executed as non-root
1. `./config.toml` on other systems

Most of the commands accept an argument to specify a custom configuration file,
allowing you to have a multiple different configurations on a single machine.
To specify a custom configuration file use the `-c` or `--config` flag, or use
the `CONFIG_FILE` environment variable.

[TOML]: https://github.com/toml-lang/toml

## Signals

It is possible to use system signals to interact with GitLab Runner. The
following commands support the following signals:

| Command | Signal | Action |
|---------|--------|--------|
| `register` | **SIGINT** | Cancel runner registration and delete if it was already registered |
| `run`, `exec`, `run-single` | **SIGINT**, **SIGTERM** | Abort all running builds and exit as soon as possible. Use twice to exit now (**forceful shutdown**). |
| `run`, `exec`, `run-single` | **SIGQUIT** | Stop accepting a new builds. Exit as soon as currently running builds do finish (**graceful shutdown**). |
| `run` | **SIGHUP** | Force to reload configuration file |

## Commands overview

This is what you see if you run `gitlab-runner` without any arguments:

```bash
NAME:
   gitlab-runner - a GitLab Runner

USAGE:
   gitlab-runner [global options] command [command options] [arguments...]

VERSION:
   1.0.0~beta.142.ga8d37f3 (a8d37f3)

AUTHOR(S):
   Kamil Trzciński <ayufan@ayufan.eu>

COMMANDS:
   exec		execute a build locally
   run		run multi runner service
   register	register a new runner
   install	install service
   uninstall	uninstall service
   start	start service
   stop		stop service
   restart	restart service
   status	get status of a service
   run-single	start single runner
   unregister	unregister specific runner
   verify	verify all registered runners
   archive	find and archive files (internal)
   artifacts	upload build artifacts (internal)
   extract	extract files from an archive (internal)
   help, h	Shows a list of commands or help for one command
```

Below we will explain what each command does in detail.

## Registration-related commands

The following commands allow you to register a new runner, or list and verify
them if they are still registered.

- [gitlab-runner register](#gitlab-runner-register)
    - [Interactive registration](#interactive-registration)
    - [Non-interactive registration](#non-interactive-registration)
- [gitlab-runner list](#gitlab-runner-list)
- [gitlab-runner verify](#gitlab-runner-verify)
- [gitlab-runner unregister](#gitlab-runner-unregister)

The above commands support the following arguments:

| Parameter   | Default | Description |
|-------------|---------|-------------|
| `--config`  | See the [configuration file section](#configuration-file) | Specify a custom configuration file to be used |

### gitlab-runner register

This command registers your GitLab Runner in GitLab. The registered runner is
added to the [configuration file](#configuration-file).
You can use multiple configurations in a single GitLab Runner. Executing
`gitlab-runner register` adds a new configuration entry, it doesn't remove the
previous ones.

There are two options to register a Runner, interactive and non-interactive.

#### Interactive registration

This command is usually used in interactive mode (**default**). You will be
asked multiple questions during a Runner's registration.

This question can be pre-filled by adding arguments when invoking the registration command:

    gitlab-runner register --name my-runner --url http://gitlab.example.com --registration-token my-registration-token

Or by configuring the environment variable before the `register` command:

    export CI_SERVER_URL=http://gitlab.example.com
    export RUNNER_NAME=my-runner
    export REGISTRATION_TOKEN=my-registration-token
    export REGISTER_NON_INTERACTIVE=true
    gitlab-runner register

To check all possible arguments and environments execute:

    gitlab-runner register --help

#### Non-interactive registration

It's possible to use registration in non-interactive / unattended mode.

You can specify the arguments when invoking the registration command:

    gitlab-runner register --non-interactive <other-arguments>

Or by configuring the environment variable before the `register` command:

    <other-environment-variables>
    export REGISTER_NON_INTERACTIVE=true
    gitlab-runner register

### gitlab-runner list

This command lists all runners saved in the
[configuration file](#configuration-file).

### gitlab-runner verify

This command checks if the registered runners can connect to GitLab, but it
doesn't verify if the runners are being used by the GitLab Runner service. An
example output is:

```bash
Verifying runner... is alive                        runner=fee9938e
Verifying runner... is alive                        runner=0db52b31
Verifying runner... is alive                        runner=826f687f
Verifying runner... is alive                        runner=32773c0f
```

To delete the old and removed from GitLab runners, execute the following
command.

>**Warning:**
This operation cannot be undone, it will update the configuration file, so
make sure to have a backup of `config.toml` before executing it.

```bash
gitlab-runner verify --delete
```

### gitlab-runner unregister

This command allows to unregister one of the registered runners. It expects to
enter a full URL and the runner's token. First get the runner's details by
executing `gitlab-runner list`:

```bash
test-runner     Executor=shell Token=t0k3n URL=http://gitlab.example.com/ci/
```

Then use this information to unregister it, using the following command.

>**Warning:**
This operation cannot be undone, it will update the configuration file, so
make sure to have a backup of `config.toml` before executing it.

```bash
gitlab-runner unregister -u http://gitlab.example.com/ci/ -t t0k3n
```

## Service-related commands

The following commands allow you to manage the runner as a system or user
service. Use them to install, uninstall, start and stop the runner service.

- [gitlab-runner install](#gitlab-runner-install)
- [gitlab-runner uninstall](#gitlab-runner-uninstall)
- [gitlab-runner start](#gitlab-runner-start)
- [gitlab-runner stop](#gitlab-runner-stop)
- [gitlab-runner restart](#gitlab-runner-restart)
- [gitlab-runner status](#gitlab-runner-status)
- [Multiple services](#multiple-services)
- [**Access Denied** when running the service-related commands](#access-denied-when-running-the-service-related-commands)

All service related commands accept these arguments:

| Parameter | Default | Description |
|-----------|---------|-------------|
| `--service-name` | `gitlab-runner` | Specify custom service name |
| `--config` | See the [configuration file](#configuration-file) | Specify a custom configuration file to use |

### gitlab-runner install

This command installs GitLab Runner as a service. It accepts different sets of
arguments depending on which system it's run on.

When run on **Windows** or as super-user, it accepts the `--user` flag which
allows you to drop privileges of builds run with the **shell** executor.

| Parameter | Default | Description |
|-----------|---------|-------------|
| `--service-name`      | `gitlab-runner` | Specify a custom name for the Runner |
| `--working-directory` | the current directory | Specify the root directory where all data will be stored when builds will be run with the **shell** executor |
| `--user`              | `root` | Specify the user which will be used to execute builds |
| `--password`          | none   | Specify the password for the user that will be used to execute the builds |

### gitlab-runner uninstall

This command stops and uninstalls the GitLab Runner from being run as an
service.

### gitlab-runner start

This command starts the GitLab Runner service.

### gitlab-runner stop

This command stops the GitLab Runner service.

### gitlab-runner restart

This command stops and then starts the GitLab Runner service.

### gitlab-runner status

This command prints the status of the GitLab Runner service.

### Multiple services

By specifying the `--service-name` flag, it is possible to have multiple GitLab
Runner services installed, with multiple separate configurations.

## Run-related commands

This command allows to fetch and process builds from GitLab.

### gitlab-runner run

This is main command that is executed when GitLab Runner is started as a
service. It reads all defined Runners from `config.toml` and tries to run all
of them.

The command is executed and works until it [receives a signal](#signals).

It accepts the following parameters.

| Parameter | Default | Description |
|-----------|---------|-------------|
| `--config`  | See [#configuration-file](#configuration-file) | Specify a custom configuration file to be used |
| `--working-directory` | the current directory | Specify the root directory where all data will be stored when builds will be run with the **shell** executor |
| `--user`    | the current user | Specify the user that will be used to execute builds |
| `--syslog`  | `false` | Send all logs to SysLog (Unix) or EventLog (Windows) |

### gitlab-runner run-single

This is a supplementary command that can be used to run only a single build
from a single GitLab instance. It doesn't use any configuration file and
requires to pass all options either as parameters or environment variables.
The GitLab URL and Runner token need to be specified too.

For example:

```bash
gitlab-runner run-single -u http://gitlab.example.com -t my-runner-token --executor docker --docker-image ruby:2.1
```

You can see all possible configuration options by using the `--help` flag:

```bash
gitlab-runner run-single --help
```

### gitlab-runner exec

This command allows you to run builds locally, trying to replicate the CI
environment as much as possible. It doesn't need to connect to GitLab, instead
it reads the local `.gitlab-ci.yml` and creates a new build environment in
which all the build steps are executed.

This command is useful for fast checking and verifying `.gitlab-ci.yml` as well
as debugging broken builds since everything is run locally.

When executing `exec` you need to specify the executor and the job name that is
present in `.gitlab-ci.yml`. The command should be executed from the root
directory of your Git repository that contains `.gitlab-ci.yml`.

`gitlab-runner exec` will clone the current state of the local Git repository.
Make sure you have committed any changes you want to test beforehand.

For example, the following command will execute the job named **tests** locally
using a shell executor:

```bash
gitlab-runner exec shell tests
```

To see a list of available executors, run:

```bash
gitlab-runner exec
```

To see a list of all available options for the `shell` executor, run:

```bash
gitlab-runner exec shell
```

If you want to use the `docker` executor with the `exec` command, use that in
context of `docker-machine shell` or `boot2docker shell`. This is required to
properly map your local directory to the directory inside the Docker container.

### Limitations of `gitlab-runner exec`

Some of the features may or may not work, like: `cache` or `artifacts`.

`gitlab-runner exec docker` can only be used when Docker is installed locally.
This is needed because GitLab Runner is using host-bind volumes to access the
Git sources.

## Internal commands

GitLab Runner is distributed as a single binary and contains a few internal
commands that are used during builds.

### gitlab-runner artifacts-downloader

Download the artifacts archive from GitLab.

### gitlab-runner artifacts-uploader

Upload the artifacts archive to GitLab.

### gitlab-runner cache-archiver

Create a cache archive, store it locally or upload it to an external server.

### gitlab-runner cache-extractor

Restore the cache archive from a locally or externally stored file.

## Troubleshooting

Below are some common pitfalls.

### **Access Denied** when running the service-related commands

Usually the [service related commands](#service-related-commands) require
administrator privileges:

- On Unix (Linux, OSX, FreeBSD) systems, prefix `gitlab-runner` with `sudo`
- On Windows systems use the elevated command prompt.
  Run an `Administrator` command prompt ([How to][prompt]).
  The simplest way is to write `Command Prompt` in the Windows search field,
  right click and select `Run as administrator`. You will be asked to confirm
  that you want to execute the elevated command prompt.
