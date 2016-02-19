## Security of running jobs

When using `gitlab-ci-multi-runner` you should be aware of potential security implications when running your jobs.

### Usage of Shell executor

**Generally it's unsafe to run tests with `shell` executors.** The jobs are run with user's permissions (gitlab-ci-multi-runner's) and can steal code from other projects that are run on this server. Use only it for running the trusted builds.

### Usage of Docker executor

**Docker can be considered safe when run in non-privileged mode.** To make such setup more secure it's advised to run jobs as user (non-root) in Docker containers with disabled sudo or dropped `SETUID` and `SETGID` capabilities.

On the other hand there's privileged mode which enables full access to host system, permission to mount and umount volumes and run nested containers. It's not advised to run containers in privileged mode.

More granular permissions can be configured in non-privileged mode via the `cap_add`/`cap_drop` settings.

## Systems with Docker installed

**This applies to installations below 0.5.0 or one's that were upgraded to newer version**

When installing package on Linux systems with Docker installed, `gitlab-ci-multi-runner` will create user that will have permisssion to access `Docker` daemon. This makes the jobs run with `shell` executor able to access `docker` with full permissions and potenially allows root access to the server.

### Usage of SSH executor

**SSH executors are susceptible to MITM attack (man-in-the-middle)**, because of missing `StrictHostKeyChecking` option. This will be fixed in one of the future releases.

### Usage of Parallels executor

**Parallels executor is the safest possible option**, because it uses full system virtualization and with VM machines that are configured to run in isolated mode it blocks access to all peripherials and shared folders.
