Configuration uses the TOML format as described here: <https://github.com/toml-lang/toml>.
The file to be edited can be found in `/home/gitlab_ci_multi_runner/config.toml`.

### The global section

This defines global settings of multi-runner.

| Setting | Explanation |
| ------- | ----------- |
| `concurrent` | limits how many jobs globally can be run concurrently. The most upper limit of jobs using all defined runners |

Example:

```bash
concurrent = 4
```

### The [[runners]] section

This defines one runner entry.

| Setting | Explanation |
| ------- | ----------- |
| `name`              | not used, just informatory |
| `url`               | CI URL |
| `token`             | runner token |
| `limit`             | limit how many jobs can be handled concurrently by this token. 0 simply means don't limit |
| `executor`          | select how a project should be built, see next section |
| `shell`             | the name of shell to generate the script (default value is platform dependent) |
| `builds_dir`        | directory where builds will be stored in context of selected executor (Locally, Docker, SSH) |
| `clean_environment` | do not inherit any environment variables from the multi-runner process |
| `environment`       | append or overwrite environment variables |
| `disable_verbose`   | don't print run commands |

Example:

```bash
[[runners]]
  name = "ruby-2.1-docker"
  url = "https://CI/"
  token = "TOKEN"
  limit = 0
  executor = "docker"
  builds_dir = ""
  shell = ""
  clean_environment = false
  environment = ["ENV=value", "LC_ALL=en_US.UTF-8"]
  disable_verbose = false
```

### The EXECUTORS

There are a couple of available executors currently.

| Executor | Explanation |
| -------- | ----------- |
| `shell`       | run build locally, default |
| `docker`      | run build using Docker container - this requires the presence of `[runners.docker]` |
| `docker-ssh`  | run build using Docker container, but connect to it with SSH - this requires the presence of `[runners.docker]` and `[runners.ssh]` |
| `ssh`         | run build remotely with SSH - this requires the presence of `[runners.ssh]` |
| `parallels`   | run build using Parallels VM, but connect to it with SSH - this requires the presence of `[runners.parallels]` and `[runners.ssh]` |

### The SHELLS

There are a couple of available shells that can be run on different platforms.

| Shell | Explanation |
| ----- | ----------- |
| `bash`        | generate Bash (Bourne-shell) script. All commands executed in Bash context (default for all Unix systems) |
| `cmd`         | generate Windows Batch script. All commands are executed in Batch context (default for Windows) |
| `powershell`  | generate Windows PowerShell script. All commands are executed in PowerShell context |

### The [runners.docker] section

This defines the Docker Container parameters.

| Parameter | Explanation |
| --------- | ----------- |
| `host`                      | specify custom Docker endpoint, by default `DOCKER_HOST` environment is used or `unix:///var/run/docker.sock` |
| `hostname`                  | specify custom hostname for Docker container |
| `tls_cert_path`             | when set it will use `ca.pem`, `cert.pem` and `key.pem` from that folder to make secure TLS connection to Docker (useful in boot2docker) |
| `image`                     | use this image to run builds |
| `privileged`                | make container run in Privileged mode (insecure) |
| `disable_cache`             | disable automatic |
| `wait_for_services_timeout` | specify how long to wait for docker services, set to 0 to disable, default: 30 |
| `cache_dir`                 | specify where Docker caches should be stored (this can be absolute or relative to current working directory) |
| `registry`                  | specify custom Docker registry to be used |
| `volumes`                   | specify additional volumes that should be cached |
| `extra_hosts`               | specify hosts that should be defined in container environment |
| `links`                     | specify containers which should be linked with building container |
| `services`                  | specify additional services that should be run with build. Please visit [Docker Registry](https://registry.hub.docker.com/) for list of available applications. Each service will be run in separate container and linked to the build. |

Example:

```bash
[runners.docker]
  host = ""
  hostname = ""
  tls_cert_path = "/Users/ayufan/.boot2docker/certs"
  image = "ruby:2.1"
  privileged = false
  disable_cache = false
  wait_for_services_timeout = 30
  cache_dir = ""
  registry = ""
  volumes = ["/data", "/home/project/cache"]
  extra_hosts = ["other-host:127.0.0.1"]
  links = ["mysql_container:mysql"]
  services = ["mysql", "redis:2.8", "postgres:9"]
```

### The [runners.parallels] section

This defines the Parallels parameters.

| Parameter | Explanation |
| --------- | ----------- |
| `base_name`         | name of Parallels VM which will be cloned |
| `template_name`     | custom name of Parallels VM linked template (optional) |
| `disable_snapshots` | if disabled the VMs will be destroyed after build |

Example:

```bash
[runners.parallels]
  base_name = "my-parallels-image"
  template_name = ""
  disable_snapshots = false
```

### The [runners.ssh] section

This defines the SSH connection parameters.

| Parameter  | Explanation |
| ---------- | ----------- |
| `host`     | where to connect (overriden when using `docker-ssh`) |
| `port`     | specify port, default: 22 |
| `user`     | specify user |
| `password` | specify password |

Example:

```
[runners.ssh]
  host = "my-production-server"
  port = "22"
  user = "root"
  password = "production-server-password"
```
