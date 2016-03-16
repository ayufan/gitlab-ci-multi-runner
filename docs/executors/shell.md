# Shell

Simple executor that allows to execute builds on local machine in context of specified user.
The shell executor supports all systems on which runner runs.
It's possible to use scripts generated for Bash, Windows PowerShell and Windows Batch.

The script can be run as unprivileged user if the `--user` is added to `gitlab-runner run` command.
This feature is only supported by Bash.

The project source is checked out to:
`<working-directory>/builds/<short-token>/<concurrent-id>/<group-name>/<project-name>`.

The caches for project are stored in
`<working-directory>/cache/<group-name>/<project-name>`.

* `<working-directory>` is the value of `--working-directory` as passed to `gitlab-runner run` command or
current directory where runner is running,
* `<short-token>` is shortened runner token, 8 first letters,
* `<concurrent-id>` is unique number, identifying the local job ID on this runner in context of this project,
* `<group-name>` is namespace where the project is stored on GitLab,
* `<project-name>` is name of the project as it is stored on GitLab

To overwrite the `<working-directory>/builds` and `<working-directory/cache` specify:
`builds_dir` and `cache_dir` in your `[[runners]]` configuration in [config.toml](../configuration/advanced_configuration.md)

## Running as unprivileged user

If the GitLab Runner is installed on Linux from `.deb` or `.rpm` package the installer will try to use `gitlab_ci_multi_runner` user if found.
If it is not found it will create a `gitlab-runner` user and use it instead.

All shell builds will be then executed as either `gitlab-runner` or `gitlab_ci_multi_runner` user.

In some testing scenarios your builds may need to access some privileged resources, like Docker Engine or VirtualBox.
You need to add the `gitlab-runner` user to group with that privileges:

    ```bash
    usermod -aG docker gitlab-runner
    usermod -aG vboxusers gitlab-runner
    ```

## Security

Generally it's unsafe to run tests with shell executors.
The jobs are run with user's permissions (gitlab-runner's) and can steal code from other projects that are run on this server.
Use only it for running the trusted builds.
