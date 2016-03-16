# VirtualBox

> Note: The *Parallels* executor works the same as *VirtualBox* executor

VirtualBox allows to use the VirtualBox virtualisation to provide a clean build environment for every build.
This executor supports all systems that can be run on VirtualBox.
The only requirement is that system needs to expose the SSH server and provide bash-compatible shell.

The project source is checked out to:
`~/builds/<group-name>/<project-name>`.

The caching is currently not supported by VirtualBox executor.

* `<group-name>` is namespace where the project is stored on GitLab,
* `<project-name>` is name of the project as it is stored on GitLab

To overwrite the `~/builds` specify:
`builds_dir` in your `[[runners]]` configuration in [config.toml](../configuration/advanced_configuration.md)

## Create a new VM

1. Import or create a new VM for VirtualBox,
2. Install OpenSSH server,
3. Install Cygwin (for Windows),
4. Install all other dependencies required by your build,
5. Shutdown the VM.

It's completely fine to use automation tools like Vagrant to provision the VirtualBox VM.

## Create a new runner

1. Install GitLab Runner on server running the VirtualBox,
2. Register GitLab Runner, select the `virtualbox` executor,
3. Put the name of current VirtualBox VM,
3. Enter SSH `user` and `password` or path to `identity_file`.

## How it works

When a new build is started:

1. Unique name for machine is generated: `runner-<short-token>-concurrent-<id>`,
1. The machine is cloned if it doesn't exist,
1. The port forward is created to access the SSH server,
1. The runner starts or restores snapshot of the VM,
1. The runner wait for the SSH server to become accessible,
1. The runner creates a snapshot of running VM (this is done to speed-up next builds),
1. The runner connects to the VM and executes a build,
1. The runner stops or shutdowns the VM.
