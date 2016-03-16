# SSH

This is simple executor that allows to execute builds on remote machine by executing command over SSH.
The SSH executor supports only scripts generated in Bash.

To use **SSH** executor you need to specify the `executor = "ssh"` and define
[runners.ssh](https://gitlab.com/gitlab-org/gitlab-ci-multi-runner/blob/master/docs/configuration/advanced-configuration.md#the-runnersssh-section).

    ```
    [[runners]]
    executor = "ssh"
    ...
    [runners.ssh]
    host = "my.server.com"
    port = "22"
    user = "root"
    password = "root.password"
    identity_file = "/path/to/identity/file"
    ```

You can use `password` or `identity_file` or both to authenticate against server.
The GitLab Runner doesn't implicitly read the file `identity_file` from `/home/user/.ssh/id_(rsa|dsa|ecdsa)`.
The `identity_file` needs to be explicitly specified.

The project source is checked out to:
`~/builds/<short-token>/<concurrent-id>/<group-name>/<project-name>`.

The caching is currently not supported for SSH executor.

* `<short-token>` is shortened runner token, 8 first letters,
* `<concurrent-id>` is unique number, identifying the local job ID on this runner in context of this project,
* `<group-name>` is namespace where the project is stored on GitLab,
* `<project-name>` is name of the project as it is stored on GitLab

To overwrite the `~/builds` specify:
`builds_dir` in your `[[runners]]` configuration in [config.toml](../configuration/advanced_configuration.md)

## Security

The SSH executor is susceptible to MITM attack (man-in-the-middle), because of missing StrictHostKeyChecking option.
This will be fixed in one of the future releases.
