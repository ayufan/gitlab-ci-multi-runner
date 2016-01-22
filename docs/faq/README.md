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
