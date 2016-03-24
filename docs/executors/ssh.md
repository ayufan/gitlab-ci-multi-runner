# SSH

>**Note:**
The SSH executor supports only scripts generated in Bash and the caching feature
is currently not supported.

This is a simple executor that allows you to execute builds on a remote machine
by executing commands over SSH.

---

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Overview](#overview)
- [Security](#security)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

To use the SSH executor you need to specify `executor = "ssh"` under the
[`[runners.ssh]`][runners-ssh] section. For example:

```toml
[[runners]]
  executor = "ssh"
  [runners.ssh]
    host = "example.com"
    port = "22"
    user = "root"
    password = "password"
    identity_file = "/path/to/identity/file"
```

You can use `password` or `identity_file` or both to authenticate against the
server. GitLab Runner doesn't implicitly read `identity_file` from
`/home/user/.ssh/id_(rsa|dsa|ecdsa)`. The `identity_file` needs to be
explicitly specified.

The project's source is checked out to:
`~/builds/<short-token>/<concurrent-id>/<namespace>/<project-name>`.

Where:

- `<short-token>` is a shortened version of the Runner's token (first 8 letters)
- `<concurrent-id>` is a unique number, identifying the local job ID on the
  particular Runner in context of the project
- `<namespace>` is the namespace where the project is stored on GitLab
- `<project-name>` is the name of the project as it is stored on GitLab

To overwrite the `~/builds` directory, specify the `builds_dir` options under
`[[runners]]` section in [`config.toml`][toml].

## Security

The SSH executor is susceptible to MITM attacks (man-in-the-middle), because of
the missing `StrictHostKeyChecking` option. This will be fixed in one of the
future releases.

[runners-ssh]: ..//configuration/advanced-configuration.md#the-runnersssh-section
[toml]: ../configuration/advanced-configuration.md
