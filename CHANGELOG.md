v 1.6.0 (unreleased)

v 1.5.0
- Update vendored toml !258
- Release armel instead arm for Debian packages !264
- Improve concurrency of docker+machine executor !254 
- Use .xz for prebuilt docker images to reduce binary size and provisioning speed of Docker Engines !249
- Remove vendored test files !271
- Update gitlab-runner-service to return 1 when no Host or PORT is defined !253
- Log caching URL address
- Retry executor preparation to reduce system failures !244
- Fix missing entrypoint script in alpine Dockerfile !248
- Suppress all but the first warning of a given type when extracting a ZIP file !261
- Mount /builds folder to all services when used with Docker Executor !272
- Cache docker client instances to avoid a file descriptor leak !260
- Support bind mount of `/builds` folder !193

v 1.4.2
- Fix abort mechanism when patching trace

v 1.4.1
- Fix panic while artifacts handling errors

v 1.4.0
- Add sentry support
- Add support for cloning VirtualBox VM snapshots as linked clones
- Add support for `security_opt` docker configuration parameter in docker executor
- Add first integration tests for executors
- Add many logging improvements (add more details to some logs, move some logs to Debug level, refactorize logger etc.)
- Make final build trace upload be done before cleanup
- Extend support for caching and artifacts to all executors
- Improve support for Docker Machine
- Improve build aborting
- Refactor common/version
- Use `environment` feature in `.gitlab-ci.yml` to track latest versions for Bleeding Edge and Stable
- Fix Absolute method for absolute path discovering for bash
- Fix zombie issues by using dumb-init instead of github.com/ramr/go-reaper

v 1.3.4
- Fix panic while artifacts handling errors

v 1.3.3
- Fix zombie issue by using dumb-init

v 1.3.2
- Fix architecture detection bug introduced in 1.3.1

v 1.3.1
- Detect architecture if not given by Docker Engine (versions before 1.9.0)

v 1.3.0
- Add incremental build trace update
- Add posibility to specify CpusetCpus, Dns and DnsSearch for docker containers created by runners
- Add a custom `User-Agent` header with version number and runtime information (go version, platform, os)
- Add artifacts expiration handling
- Add artifacts handling for failed builds
- Add customizable `check_interval` to set how often to check GitLab for a new builds
- Add docker Machine IP address logging
- Make Docker Executor ARM compatible
- Refactor script generation to make it fully on-demand
- Refactor runnsers Acquire method to improve performance
- Fix branch name setting at compile time
- Fix panic when generating log message if provision of node fails
- Fix docker host logging
- Prevent leaking of goroutines when aborting builds
- Restore valid version info in --help message
- [Experimental] Add `GIT_STRATEGY` handling - clone/fetch strategy configurable per job
- [Experimental] Add `GIT_DEPTH` handling - `--depth` parameter for `git fetch` and `git clone`

v 1.2.0
- Use Go 1.6
- Add `timeout` option for the `exec` command
- Add runtime platform information to debug log
- Add `docker-machine` binary to Runner's official docker images
- Add `build_current` target to Makefile - to build only a binary for used architecture
- Add support for `after_script`
- Extend version information when using `--version` flag
- Extend artifacts download/upload logs with more response data
- Extend unregister command to accept runner name
- Update shell detection mechanism
- Update the github.com/ayufan/golag-kardianos-service dependency
- Replace ANSI_BOLD_YELLOW with ANSI_YELLOW color for logging
- Reconcile VirtualBox status constants with VBoxManage output values
- Make checkout quiet
- Make variables to work at job level in exec mode
- Remove "user mode" warning when running in a system mode
- Create `gitlab-runner` user as a system account
- Properly create `/etc/gitlab-runner/certs` in Runner's official docker images
- Disable recursive submodule fetchin on fetching changes
- Fix nil casting issue on docker client creation
- Fix used build platforms for `gox`
- Fix a limit problems when trying to remove a non-existing machines
- Fix S3 caching issues
- Fix logging messages on artifacts dowloading
- Fix binary panic while using VirtualBox executor with no `vboxmanage` binary available

v 1.1.0
- Use Go 1.5
- Change license to MIT
- Add docker-machine based auto-scaling for docker executor
- Add support for external cache server
- Add support for `sh`, allowing to run builds on images without the `bash`
- Add support for passing the artifacts between stages
- Add `docker-pull-policy`, it removes the `docker-image-ttl`
- Add `docker-network-mode`
- Add `git` to gitlab-runner:alpine
- Add support for `CapAdd`, `CapDrop` and `Devices` by docker executor
- Add support for passing the name of artifacts archive (`artifacts:name`)
- Add support for running runner as system service on OSX
- Refactor: The build trace is now implemented by `network` module
- Refactor: Remove CGO dependency on Windows
- Fix: Create alternative aliases for docker services (uses `-`)
- Fix: VirtualBox port race condition
- Fix: Create cache for all builds, including tags
- Fix: Make the shell executor more verbose when the process cannot be started
- Fix: Pass gitlab-ci.yml variables to build container created by docker executor
- Fix: Don't restore cache if not defined in gitlab-ci.yml
- Fix: Always use `json-file` when starting docker containers
- Fix: Error level checking for Windows Batch and PowerShell

v 1.0.4
- Fix support for Windows PowerShell

v 1.0.3
- Fix support for Windows Batch
- Remove git index lock file: this solves problem with git checkout being terminated
- Hijack docker.Client to use keep-alives and to close extra connections

v 1.0.2
- Fix bad warning about not found untracked files
- Don't print error about existing file when restoring the cache
- When creating ZIP archive always use forward-slashes and don't permit encoding absolute paths
- Prefer to use `path` instead of `filepath` which is platform specific: solves the docker executor on Windows

v 1.0.1
- Use nice log formatting for command line tools
- Don't ask for services during registration (we prefer the .gitlab-ci.yml)
- Create all directories when extracting the file

v 1.0.0
- Add `gitlab-runner exec` command to easy running builds
- Add `gitlab-runner status` command to easy check the status of the service
- Add `gitlab-runner list` command to list all runners from config file
- Allow to specify `ImageTTL` for configuration the frequency of docker image re-pulling (see advanced-configuration)
- Inject TLS certificate chain for `git clone` in build container, the gitlab-runner SSL certificates are used
- Remove TLSSkipVerify since this is unsafe option
- Add go-reaper to make gitlab-runner to act as init 1 process fixing zombie issue when running docker container
- Create and send artifacts as zip files
- Add internal commands for creating and extracting archives without the system dependencies
- Add internal command for uploading artifacts without the system dependencies
- Use umask in docker build containers to fix running jobs as specific user
- Fix problem with `cache` paths never being archived
- Add support for [`cache:key`](http://doc.gitlab.com/ce/ci/yaml/README.html#cachekey)
- Add warnings about using runner in `user-mode`
- Push packages to all upcoming distributions (Debian/Ubuntu/Fedora)
- Rewrite the shell support adding all features to all shells (makes possible to use artifacts and caching on Windows)
- Complain about missing caching and artifacts on some executors
- Added VirtualBox executor
- Embed prebuilt docker build images in runner binary and load them if needed
- Make possible to cache absolute paths (unsafe on shell executor)

v 0.7.2
- Adjust `umask` for build image
- Use absolute path when executing archive command
- Fix regression when variables were not passed to service container
- Fix duplicate files in cache or artifacts archive

v 0.7.1
- Fix caching support
- Suppress tar verbose output

v 0.7.0
- Refactor code structure
- Refactor bash script adding pre-build and post-build steps
- Add support for build artifacts
- Add support for caching build directories
- Add command to generate archive with cached folders or artifacts
- Use separate containers to run pre-build (git cloning), build (user scripts) and post-build (uploading artifacts)
- Expand variables, allowing to use $CI_BUILD_TAG in image names, or in other variables
- Make shell executor to use absolute path for project dir
- Be strict about code formatting
- Move network related code to separate package
- Automatically load TLS certificates stored in /etc/gitlab-runner/certs/<hostname>.crt
- Allow to specify tls-ca-file during registration
- Allow to disable tls verification during registration

v 0.6.1
- Revert: Fix tags handling when using git fetch: fetch all tags and prune the old ones

v 0.6.0
- Fetch docker auth from ~/.docker/config.json or ~/.dockercfg
- Added support for NTFSSecurity PowerShell module to address problems with long paths on Windows
- Make the service startup more readable in case of failure: print a nice warning message
- Command line interface for register and run-single accepts all possible config parameters now
- Ask about tags and fix prompt to point to gitlab.com/ci
- Pin to specific Docker API version
- Fix docker volume removal issue
- Add :latest to imageName if missing
- Pull docker images every minute
- Added support for SIGQUIT to allow to gracefully finish runner: runner will not accept new jobs, will stop once all current jobs are finished.
- Implicitly allow images added as services
- Evaluate script command in subcontext, making it to close stdin (this change since 0.5.x where the separate file was created)
- Pass container labels to docker
- Force to use go:1.4 for building packages
- Fix tags handling when using git fetch: fetch all tags and prune the old ones
- Remove docker socket from gitlab/gitlab-runner images
- Pull (update) images and services every minute
- Ignore options from Coordinator that are null
- Provide FreeBSD binary
- Use -ldflags for versioning
- Update go packages
- Fix segfault on service checker container
- WARNING: By default allow to override image and services

v 0.5.5
- Fix cache_dir handling

v 0.5.4
- Update go-dockerclient to fix problems with creating docker containers

v 0.5.3
- Pin to specific Docker API version
- Fix docker volume removal issue

v 0.5.2
- Fixed CentOS6 service script
- Fixed documentation
- Added development documentation
- Log service messages always to syslog

v 0.5.1
- Update link for Docker configuration

v 0.5.0
- Allow to override image and services for Docker executor from Coordinator
- Added support for additional options passed from coordinator
- Added support for receiving and defining allowed images and services from the Coordinator
- Rename gitlab_ci_multi_runner to gitlab-runner
- Don't require config file to exist in order to run runner
- Change where config file is stored: /etc/gitlab-runner/config.toml (*nix, root), ~/.gitlab-runner/config.toml (*nix, user)
- Create config on service install
- Require root to control service on Linux
- Require to specify user when installing service
- Run service as root, but impersonate as --user when executing shell scripts
- Migrate config.toml from user directory to /etc/gitlab-runner/
- Simplify service installation and upgrade
- Add --provides and --replaces to package builder
- Powershell: check exit code in writeCommandChecked
- Added installation tests
- Add runner alpine-based image
- Send executor features with RunnerInfo
- Verbose mode by using `echo` instead of `set -v`
- Colorize bash output
- Set environment variables from bash script: this fixes problem with su
- Don't cache Dockerfile VOLUMEs
- Pass (public) environment variables received from Coordinator to service containers

v 0.4.2
- Force GC cycle after processing build
- Use log-level set to info, but also make `Checking for builds: nothing` being print as debug
- Fix memory leak - don't track references to builds

v 0.4.1
- Fixed service reregistration for RedHat systems

v 0.4.0
- Added CI=true and GITLAB_CI=true to environment variables
- Added output_limit (in kilobytes) to runner config which allows to enlarge default build log size
- Added support for custom variables received from CI
- Added support for SSH identity file
- Optimize build path to make it shorter, more readable and allowing to fix shebang issue
- Make the debug log human readable
- Make default build log limit set to 4096 (4MB)
- Make default concurrent set to 1
- Make default limit for runner set to 1 during registration
- Updated kardianos service to fix OSX service installation
- Updated logrus to make console output readable on Windows
- Change default log level to warning
- Make selection of forward or back slashes dependent by shell not by system
- Prevent runner to be stealth if we reach the MaxTraceOutputSize
- Fixed Windows Batch script when builds are located on different drive
- Fixed Windows runner
- Fixed installation scripts path
- Fixed wrong architecture for i386 debian packages
- Fixed problem allowing commands to consume build script making the build to succeed even if not all commands were executed

v 0.3.4
- Create path before clone to fix Windows issue
- Added CI=true and GITLAB_CI=true
- Fixed wrong architecture for i386 debian packages

v 0.3.3
- Push package to ubuntu/vivid and ol/6 and ol/7

v 0.3.2
- Fixed Windows batch script generator

v 0.3.1
- Remove clean_environment (it was working only for shell scripts)
- Run bash with --login (fixes missing .profile environment)

v 0.3.0
- Added repo slug to build path
- Build path includes repository hostname
- Support TLS connection with Docker
- Default concurrent limit is set to number of CPUs
- Make most of the config options optional
- Rename setup/delete to register/unregister
- Checkout as detached HEAD (fixes compatibility with older git versions)
- Update documentation

v 0.2.0
- Added delete and verify commands
- Limit build trace size (1MB currently)
- Validate build log to contain only valid UTF-8 sequences
- Store build log in memory
- Integrate with ci.gitlab.com
- Make packages for ARM and CentOS 6 and provide beta version
- Store Docker cache in separate containers
- Support host-based volumes for Docker executor
- Don't send build trace if nothing changed
- Refactor build class

v 0.1.17
- Fixed high file descriptor usage that could lead to error: too many open files

v 0.1.16
- Fixed systemd service script

v 0.1.15
- Fix order of executor commands
- Fixed service creation options
- Fixed service installation on OSX

v 0.1.14
- Use custom kardianos/service with enhanced service scripts
- Remove all system specific packages and use universal for package manager

v 0.1.13
- Added abstraction over shells
- Moved all bash specific stuff to shells/bash.go
- Select default shell for OS (bash for Unix, batch for Windows)
- Added Windows Cmd support
- Added Windows PowerShell support
- Added the kardianos/service which allows to easily run gitlab-ci-multi-runner as service on different platforms
- Unregister Parallels VMs which are invalid
- Delete Parallels VM if it doesn't contain snapshots
- Fixed concurrency issue when assigning unique names

v 0.1.12
- Abort all jobs if interrupt or SIGTERM is received
- Runner now handles HUP and reloads config on-demand
- Refactored runner setup allowing to non-interactive configuration of all questioned parameters
- Added CI_PROJECT_DIR environment variable
- Make golint happy (in most cases)

v 0.1.11
- Package as .deb and .rpm and push it to packagecloud.io (for now)

v 0.1.10
- Wait for docker service to come up (Loïc Guitaut)
- Send build log as early as possible

v 0.1.9
- Fixed problem with resetting ruby environment

v 0.1.8
- Allow to use prefixed services
- Allow to run on Heroku
- Inherit environment variables by default for shell scripts
- Mute git messages during checkout
- Remove some unused internal messages from build log

v 0.1.7
- Fixed git checkout

v 0.1.6
- Remove Docker containers before starting job

v 0.1.5
- Added Parallels executor which can use snapshots for fast revert (only OSX supported)
- Refactored sources

v 0.1.4
- Remove Job and merge it into Build
- Introduce simple API server
- Ask for services during setup

v 0.1.3
- Optimize setup
- Optimize multi-runner setup - making it more concurrent
- Send description instead of hostname during registration
- Don't ask for tags

v 0.1.2
- Make it work on Windows

v 0.1.1
- Added Docker services

v 0.1.0
- Initial public release
