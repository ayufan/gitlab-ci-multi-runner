# GitLab Runner Commands

GitLab Runner contains of set commands that allows to register, manage and run your builds.

## Preface

### Checking commands help

You can check a recent list of commands by executing:

    gitlab-runner --help

To see a help for specific command append `--help` after a command:

    gitlab-runner command --help

### Using environment variables

Most of commands supports environment variables as a method to pass configuration to command.

The name of environment variable is present when checking the help for command:

        $ gitlab-runner run --help
        NAME:
           gitlab-runner run - run multi runner service
        
        USAGE:
           gitlab-runner run [command options] [arguments...]
        
        OPTIONS:
           -c, --config "/Users/ayufan/.gitlab-runner/config.toml"	Config file [$CONFIG_FILE]

### Running in debug mode

It's possible to run GitLab Runner in debug mode.
Debug mode is especially useful when looking for the cause of undefined behavior or error.

To run command in debug mode prepend the command with `--debug`:

    $ gitlab-runner --debug command

### Super-user permission

Commands that access GitLab Runner configuration behave differently when executed as super-user (root).
The file location is dependent on the user executing the command.

Be aware of the notice that is written when executing some of these commands:
running builds, registering services, managing registered runners.

    $ gitlab-runner run                                                                                                                                                                                                                                      [20:18:03]
    INFO[0000] Starting multi-runner from /Users/ayufan/.gitlab-runner/config.toml ...  builds=0
    WARN[0000] Running in user-mode.                        
    WARN[0000] Use sudo for system-mode:                    
    WARN[0000] $ sudo gitlab-runner...    

You should use `user-mode` if you are really sure that this is a mode that you want to work in.
Otherwise prefix your command with `sudo`:

    $ sudo gitlab-runner run
    INFO[0000] Starting multi-runner from /etc/gitlab-runner/config.toml ...  builds=0
    INFO[0000] Running in system-mode.

In case of **Windows** you may need to run **Command Prompt** in **Administrative Mode**.

### Configuration file

GitLab Runner configuration uses the [TOML][] format.

The file to be edited can be found in:

1. `/etc/gitlab-runner/config.toml` on *nix systems when gitlab-runner is
   executed as super-user (root). **This is also path for service configuration**
1. `~/.gitlab-runner/config.toml` on *nix systems when gitlab-runner is
   executed as non-root,
1. `./config.toml` on other systems

Most of the commands accepts argument to specify custom localisation of configuration file,
allowing to have a multiple different configurations on single machine.
To specify custom configuration use:
`-c` or `--config` as an parameter,
or use `CONFIG_FILE` environment variable.

### Signals

It is possible to use system signals to interact with GitLab Runner.
Usually to use signals you can can do (it's not the safest way, because all gitlab-runner processes receive this signal):

    killall -SIGQUIT gitlab-runner

These commands supports a following signals:

| Command | Signal | Action |
|---------|--------|--------|
| **register** | **SIGINT** | Cancel runner registration and delete if it was already registered |
| **run**, **exec**, **run-single** | **SIGINT**, **SIGTERM** | Abort all running builds and exit as soon as possible. Use twice to exit now (__forceful shutdown__). |
| **run**, **exec**, **run-single** | **SIGQUIT** | Stop accepting a new builds. Exit as soon as currently running builds do finish (__graceful shutdown__). |
| **run** | SIGHUP | Force to reload configuration file |

## Commands

    $ gitlab-runner
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

### Registration-related methods

This methods allows you to register a new runner, list and verify them if they are still registered.

All commands supports the following arguments:

| Parameter | Default | Description |
|-----------|---------|-------------|
| --config | See the #ConfigurationFile | Specify custom configuration file to be used |

#### gitlab-runner register

This command registers your GitLab Runner in GitLab.
**You can register multiple configurations in single GitLab Runner.**
**Executing `gitlab-runner register` does add a new configuration, and it doesn't remove the previous ones.**

The registered runner is added to [Configuration File](#ConfigurationFile).

##### Interactive registration

This command is usually used in interactive mode (**default**).
You will be asked a multiple questions during registration.

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

##### Non-interactive registration

It's possible to use registration in non-interactive / unattended mode.

You can specify the arguments when invoking the registration command:

    gitlab-runner register --non-interactive <other-arguments>

Or by configuring the environment variable before the `register` command:

    <other-environment-variables>
    export REGISTER_NON_INTERACTIVE=true
    gitlab-runner register

#### gitlab-runner list

This command lists all runners saved in [Configuration File](#ConfigurationFile).

#### gitlab-runner verify

This command checks if the registered runners can connect to GitLab and if they **registered** in GitLab.

**This command doesn't verify if they are being used by the GitLab Runner service.**

Executing `gitlab-runner verify --delete` it is possible to delete old, and removed from GitLab runners.
This will update the configuration file. 

#### gitlab-runner unregister

This command allows to unregister one of the registered runners.

The command expects to enter a full URL and runner token.
You can get a runner details by executing the `gitlab-runner list`

    $ gitlab-runner list
    test-runner     Executor=shell Token=my-token URL=http://gitlab.example.com/ci/
    $ gitlab-runner unregister -u http://gitlab.example.com/ci/ -t my-token
    Deleting runner... succeeded
    Updated /etc/gitlab-runner/config.toml

### Service-related methods

This methods allows you to manage runner as system or user service.
Use them to install, uninstall, start and stop the runner service.

All service related methods accepts these arguments:

| Parameter | Default | Description |
|-----------|---------|-------------|
| --service-name | gitlab-runner | Specify custom service name |
| --config | See the #ConfigurationFile | Specify custom configuration file to use by service |

#### gitlab-runner install

This commands install GitLab Runner as an service.
It accepts different set of arguments depending on which system it's run.

When run on **Windows** or as super-user it accepts `--user` which allows you to drop privileges
of builds run with **shell** executor.

| Parameter | Default | Description |
|-----------|---------|-------------|
| --service-name | gitlab-runner | Specify custom 
| --working-directory | (current directory) | Specify root directory where all data will be stored when builds will be run with **shell** executor |
| --user | root | Specify user which will be used to execute builds |
| --password | _none_ | Specify password for user that is used to execute builds (**Windows**-only) |

#### gitlab-runner uninstall

This commands stops and uninstall GitLab Runner from being run as an service.

#### gitlab-runner start

This commands starts GitLab Runner service.

#### gitlab-runner stop

This commands stops GitLab Runner service.

#### gitlab-runner restart

This commands stops and starts GitLab Runner service.

#### gitlab-runner status

This commands prints the status of GitLab Runner service.

#### Multiple services

By specifying the `--service-name` is it possible to have multiple GitLab Runner services installed,
with multiple separate configurations.

#### **Access Denied** when running the service-related methods

Usually the service related methods required administrator permission:

- on Unix (Linux, OSX, FreeBSD) systems prefix the `gitlab-runner` with `sudo`.
- on Windows system use the elevated command prompt.
Run an `Administrator` command prompt ([How to][prompt]).
The simplest is to write `Command Prompt` in Windows search field,
right click and select `Run as administrator`.
You will be asked to confirm that you want to execute the elevated command prompt. 

### Run-related methods

This methods allow to fetch and process builds from GitLab.

#### gitlab-runner run

This is main method that is executed when GitLab Runner is started as an service.
It reads all defined runners from configuration and tries to run all of them.

The command is executed and works till it receives an signal.

This method accepts the following arguments:

| Parameter | Default | Description |
|-----------|---------|-------------|
| --config | See the #ConfigurationFile | Specify custom configuration file to be used |
| --working-directory | (current directory) | Specify root directory where all data will be stored when builds will be run with **shell** executor |
| --user | (current user) | Specify user which will be used to execute builds |
| --syslog | false | Send all logs to SysLog (Unix) or EventLog (Windows) |

#### gitlab-runner run-single

This is supplementary command that can be used to run only single build from single GitLab.
This command doesn't use any configuration file and requires to pass all options either with arguments or environment variables.
The GitLab URL and Runner Token needs to be specified too.

    gitlab-runner run-single -u http://gitlab.example.com -t my-runner-token --executor docker --docker-image ruby:2.1

All possible configuration options can be seen by adding `--help`:

    gitlab-runner run-single --help

#### gitlab-runner exec <executor> <job>

This is command that allows to run builds locally, trying to replicate the CI environment as much as possible.
This methods doesn't connect to GitLab server, instead it reads local `.gitlab-ci.yml` and creates a new
build environment in which all build steps are executed.

This command is useful for fast checking and verifying the `.gitlab-ci.yml` and trying to debug broken builds
since everything is run locally.

When executing `exec` you need to specify `executor` and `job name` that is present in `.gitlab-ci.yml`.
The command should be executed from root directory of your git repository.

    gitlab-runner exec shell tests

This will execute job **tests** locally using a shell executor.

To see a list of available executors:

    gitlab-runner exec

To see a list of all available options for executor:

    gitlab-runner exec shell

If you want to use **docker** executor with that command, use that in context of **docker-machine shell** or **boot2docker shell**.
This is required to properly map your local directory to directory in **docker**.

#### Limitations of `gitlab-runner exec`

Some of the features may or may not work, like: caching or artifacts.

### Internal commands

The GitLab Runner is distributed as single binary and contains a few internal commands that are used during builds.

#### gitlab-runner archive

Create a cache or artifacts archive.

#### gitlab-runner artifacts

Upload the artifacts archive to GitLab.

#### gitlab-runner extract

Extract a cache archive in context of current build.
