# VirtualBox

> Note: The Parallels executor works the same as the VirtualBox executor. The
caching feature is currently not supported.

VirtualBox allows you to use VirtualBox's virtualization to provide a clean
build environment for every build. This executor supports all systems that can
be run on VirtualBox. The only requirement is that the virtual machine exposes
its SSH server and provide a bash-compatible shell.

---

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Overview](#overview)
- [Create a new base virtual machine](#create-a-new-base-virtual-machine)
- [Create a new Runner](#create-a-new-runner)
- [How it works](#how-it-works)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

The project's source code is checked out to: `~/builds/<namespace>/<project-name>`.

Where:

- `<namespace>` is the namespace where the project is stored on GitLab
- `<project-name>` is the name of the project as it is stored on GitLab

To overwrite the `~/builds` directory, specify the `builds_dir` option under
the `[[runners]]` section in
[`config.toml`](../configuration/advanced-configuration.md).

## Create a new base virtual machine

1. Import or create a new virtual machine in VirtualBox
1. Log into the new virtual machine
1. If on Windows, install [Cygwin]
1. Install the OpenSSH server
1. Install all other dependencies required by your build
1. Log out and shutdown the virtual machine

It's completely fine to use automation tools like Vagrant to provision the
virtual machine.

## Create a new Runner

1. Install GitLab Runner on the host running VirtualBox
1. Register a new GitLab Runner with `gitlab-ci-multi-runner register`
1. Select the `virtualbox` executor
1. Enter the name of the base virtual machine you created earlier (find it under
   the settings of the virtual machine **General > Basic > Name**)
1. Enter the SSH `user` and `password` or path to `identity_file` of the
   virtual machine

## How it works

When a new build is started:

1. A unique name for the virtual machine is generated: `runner-<short-token>-concurrent-<id>`
1. The virtual machine is cloned if it doesn't exist
1. The port forward rules are created to access the SSH server
1. The Runner starts or restores the snapshot of the virtual machine
1. The Runner waits for the SSH server to become accessible
1. The Runner creates a snapshot of the running virtual machine (this is done
   to speed up any next builds)
1. The Runner connects to the virtual machine and executes a build
1. The Runner stops or shutdowns the virtual machine

[cygwin]: https://cygwin.com/
