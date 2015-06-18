v 0.4.0
- Added CI=true and GITLAB_CI=true to environment variables
- Added output_limit (in kilobytes) to runner config which allows to enlarge default build log size
- Added support for custom variables received from CI
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
