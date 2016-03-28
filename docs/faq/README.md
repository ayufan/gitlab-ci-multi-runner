# FAQ

## 1. Where are logs stored for service?

+ If the GitLab Runner is run as service on Linux/OSX  the daemon logs to syslog.
+ If the GitLab Runner is run as service on Windows it logs to System's Event Log.

## 2. Run in `--debug` mode

Is it possible to run GitLab Runner in debug/verbose mode. Do it from terminal:

```
gitlab-runner --debug run
```

## 3. I get a PathTooLongException during my builds on Windows

This is caused by tools like `npm` which will sometimes generate directory structures
with paths more than 260 characters in length. There are two possible fixes you can
adopt to solve the problem.

### a) Use Git with core.longpaths enabled

You can avoid the problem by using Git to clean your directory structure, first run
`git config --system core.longpaths true` from the command line and then set your
project to use *git fetch* from the GitLab CI project settings page.

### b) Use NTFSSecurity tools for PowerShell

The [NTFSSecurity](https://ntfssecurity.codeplex.com/) PowerShell module provides
a *Remove-Item2* method which supports long paths. The Gitlab CI Multi Runner will
detect it if it is available and automatically make use of it.

## 4. I'm seeing `x509: certificate signed by unknown authority`

Please [See the self-signed certificates](../configuration/tls-self-signed.md)

## 5. I get `Permission Denied` when accessing the `/var/run/docker.sock`

If you want to use Docker executor,
and you are connecting to Docker Engine installed on server.
You can see the `Permission Denied` error.
The most likely cause is that your system uses SELinux (enabled by default on CentOS, Fedora and RHEL).
Check your SELinux policy on your system for possible denials.

## 6. The Docker executor gets timeout when building Java project.

This most likely happens, because of the broken AUFS storage driver:
[Java process hangs on inside container](https://github.com/docker/docker/issues/18502).
The best solution is to change the [storage driver](https://docs.docker.com/engine/userguide/storagedriver/selectadriver/)
to either OverlayFS (faster) or DeviceMapper (slower).

Check this article about [configuring and running Docker](https://docs.docker.com/engine/articles/configuring/)
or this article about [control and configure with systemd](https://docs.docker.com/engine/articles/systemd/).

## 7. I get 411 when uploading artifacts.

This happens due to fact that runner uses `Transfer-Encoding: chunked` which is broken on early version of Nginx (http://serverfault.com/questions/164220/is-there-a-way-to-avoid-nginx-411-content-length-required-errors).

Upgrade your Nginx to newer version. For more information see this issue: https://gitlab.com/gitlab-org/gitlab-ci-multi-runner/issues/1031

## 8. I can't run Windows BASH scripts; I'm getting `The system cannot find the batch label specified - buildscript`.

You need to prepend `call` to your batch file line in .gitlab-ci.yml so that it looks like `call C:\path\to\test.bat`. Here
is a more complete example:

```
before_script:
  - call C:\path\to\test.bat
```

Additional info can be found under issue [#1025](https://gitlab.com/gitlab-org/gitlab-ci-multi-runner/issues/1025).

## 9. My gitlab runner is on Windows. How can I get colored ouptut on the web terminal?

**Short answer:**

Make sure that you have the ANSI color codes in your program's output. For the purposes of text formatting, assume that you're
running in a UNIX ANSI terminal emulator (because that's what the webUI's output is).

**Long Answer:**

The web interface for gitlab-ci emulates a UNIX ANSI terminal (at least partially). The `gitlab-runner` pipes any output from the build
directly to the web interface. That means that any ANSI color codes that are present will be honored.

Windows' CMD terminal (before Win10 ([source](http://www.nivot.org/blog/post/2016/02/04/Windows-10-TH2-(v1511)-Console-Host-Enhancements)))
does not support ANSI color codes - it uses win32 ([`ANSI.SYS`](https://en.wikipedia.org/wiki/ANSI.SYS)) calls instead which are **not** present in
the string to be displayed. When writing cross-platform programs, a developer will typically use ANSI color codes by default and convert
them to win32 calls when running on a Windows system (example: [Colorama](https://pypi.python.org/pypi/colorama)).

If you're program is doing the above, then you need to disable that conversion for the CI builds so that the ANSI codes remain in the string.

See issue [#332](https://gitlab.com/gitlab-org/gitlab-ci-multi-runner/issues/332) for more information.

## 10. "warning: You appear to have cloned an empty repository."

When running `git clone` using HTTP(s) (with GitLab Runner or manually for
tests) you have received an output:

```bash
$ git clone https://git.example.com/user/repo.git

Cloning into 'repo'...
warning: You appear to have cloned an empty repository.
```

Make sure, that configuration of the HTTP Proxy in your GitLab server
installation is done properly. Especially if you are using some HTTP Proxy with
its own configuration, make sure that GitLab requests are proxied to the
**GitLab Workhorse socket**, not to the **GitLab unicorn socket**.

Git protocol via HTTP(S) is resolved by the GitLab Workhorse, so this is the
**main entrypoint** of GitLab.

If you are using Omnibus GitLab, but don't want to use the bundled Nginx
server, please read [using a non-bundled web-server][omnibus-ext-nginx].

In gitlab-recipes repository there are [web-server configuration
examples][recipes] for Apache and Nginx.

If you are using GitLab installed from source, also please read the above
documentation and examples, and make sure that all HTTP(S) traffic is going
trough the **GitLab Workhorse**.

See [an example of a user issue][1105].

[omnibus-ext-nginx]: http://doc.gitlab.com/omnibus/settings/nginx.html#using-a-non-bundled-web-server
[recipes]: https://gitlab.com/gitlab-org/gitlab-recipes/tree/master/web-server
[1105]: https://gitlab.com/gitlab-org/gitlab-ci-multi-runner/issues/1105

## 11. `"launchctl" failed: exit status 112, Could not find domain for`

This message may occur when you try to install GitLab Runner on OSX. Make sure
that you manage GitLab Runner service from the GUI Terminal application, not
the SSH connection.

## 12. `Failed to authorize rights (0x1) with status: -60007.`

If your Runner is stuck on the above message when using OSX, there are two
problems why this happens:

1. Make sure that your user can perform UI interactions:

    ```bash
    DevToolsSecurity -enable
    sudo security authorizationdb remove system.privilege.taskport is-developer
    ```

    The first command enables access to developer tools for your user.
    The second command allows the user who is member of the developer group to
    do UI interactions, e.g., run the iOS simulator.

    ---

2. Make sure that your Runner service doesn't use `SessionCreate = true`.
   Previously, when running GitLab Runner as a service, we were creating
   `LaunchAgents` with `SessionCreate`. At that point (**Mavericks**), this was
   the only solution to make Code Signing work. That changed recently with
   **OSX El Capitan** which introduced a lot of new security features that
   altered this behavior.
   Since GitLab Runner 1.1, when creating a `LaunchAgent`, we don't set
   `SessionCreate`. However, in order to upgrade, you need to manually
   reinstall the `LaunchAgent` script:

    ```
    gitlab-ci-multi-runner uninstall
    gitlab-ci-multi-runner install
    gitlab-ci-multi-runner start
    ```

    Then you can verify that `~/Library/LaunchAgents/gitlab-runner.plist` has
    `SessionCreate` set to `false`.
