# FAQ

## 1. Where are logs stored for service?

+ If the GitLab Runner is run as service on Linux/OSX  the daemon logs to syslog.
+ If the GitLab Runner is run as service on Windows it logs to System's Event Log.

## 2. Run in `--debug` mode

Is it possible to run GitLab Runner in debug/verbose mode. Do it from terminal:

```
gitlab-runner --debug run
```
