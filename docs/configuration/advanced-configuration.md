# Advanced configuration

GitLab Runner configuration uses the [TOML][] format.

The file to be edited can be found in:

1. `/etc/gitlab-runner/config.toml` on *nix systems when gitlab-runner is
   executed as root. **This is also path for service configuration**
1. `~/.gitlab-runner/config.toml` on *nix systems when gitlab-runner is
   executed as non-root,
1. `./config.toml` on other systems

## The global section

This defines global settings of multi-runner.

| Setting | Description |
| ------- | ----------- |
| `concurrent` | limits how many jobs globally can be run concurrently. The most upper limit of jobs using all defined runners |

Example:

```bash
concurrent = 4
```

## The [[runners]] section

This defines one runner entry.

| Setting | Description |
| ------- | ----------- |
| `name`              | not used, just informatory |
| `url`               | CI URL |
| `token`             | runner token |
| `tls-ca-file`       | file containing the certificates to verify the peer when using HTTPS |
| `tls-skip-verify`   | whether to verify the TLS certificate when using HTTPS, default: false |
| `limit`             | limit how many jobs can be handled concurrently by this token. 0 simply means don't limit |
| `executor`          | select how a project should be built, see next section |
| `shell`             | the name of shell to generate the script (default value is platform dependent) |
| `builds_dir`        | directory where builds will be stored in context of selected executor (Locally, Docker, SSH) |
| `cache_dir`         | directory where build caches will be stored in context of selected executor (Locally, Docker, SSH) |
| `environment`       | append or overwrite environment variables |
| `disable_verbose`   | don't print run commands |
| `output_limit`      | set maximum build log size in kilobytes, by default set to 4096 (4MB) |

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
  environment = ["ENV=value", "LC_ALL=en_US.UTF-8"]
  disable_verbose = false
```

## The EXECUTORS

There are a couple of available executors currently.

| Executor | Description |
| -------- | ----------- |
| `shell`       | run build locally, default |
| `docker`      | run build using Docker container - this requires the presence of `[runners.docker]` and [Docker Engine][] installed on the system that the Runner runs |
| `docker-ssh`  | run build using Docker container, but connect to it with SSH - this requires the presence of `[runners.docker]` , `[runners.ssh]` and [Docker Engine][] installed on the system that the Runner runs |
| `ssh`         | run build remotely with SSH - this requires the presence of `[runners.ssh]` |
| `parallels`   | run build using Parallels VM, but connect to it with SSH - this requires the presence of `[runners.parallels]` and `[runners.ssh]` |

## The SHELLS

There are a couple of available shells that can be run on different platforms.

| Shell | Description |
| ----- | ----------- |
| `bash`        | generate Bash (Bourne-shell) script. All commands executed in Bash context (default for all Unix systems) |
| `cmd`         | generate Windows Batch script. All commands are executed in Batch context (default for Windows) |
| `powershell`  | generate Windows PowerShell script. All commands are executed in PowerShell context |

## The [runners.docker] section

This defines the Docker Container parameters.

| Parameter | Description |
| --------- | ----------- |
| `host`                      | specify custom Docker endpoint, by default `DOCKER_HOST` environment is used or `unix:///var/run/docker.sock` |
| `hostname`                  | specify custom hostname for Docker container |
| `tls_cert_path`             | when set it will use `ca.pem`, `cert.pem` and `key.pem` from that folder to make secure TLS connection to Docker (useful in boot2docker) |
| `image`                     | use this image to run builds |
| `privileged`                | make container run in Privileged mode (insecure) |
| `disable_cache`             | disable automatic |
| `wait_for_services_timeout` | specify how long to wait for docker services, set to 0 to disable, default: 30 |
| `cache_dir`                 | specify where Docker caches should be stored (this can be absolute or relative to current working directory) |
| `volumes`                   | specify additional volumes that should be mounted (same syntax as Docker -v option) |
| `extra_hosts`               | specify hosts that should be defined in container environment |
| `links`                     | specify containers which should be linked with building container |
| `services`                  | specify additional services that should be run with build. Please visit [Docker Registry](https://registry.hub.docker.com/) for list of available applications. Each service will be run in separate container and linked to the build. |
| `allowed_images`            | specify wildcard list of images that can be specified in .gitlab-ci.yml |
| `allowed_services`          | specify wildcard list of services that can be specified in .gitlab-ci.yml |
| `shared_builds_dir`         | run each build in a different subdirectory of the builds directory (useful if `/builds` is shared between containers with a volume. ) |
| `disable_build_volume`      | don't create a volume that contains the cloned repository. You should only use this if you are going to create your own volume in the builds directory. |

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
  volumes = ["/data", "/home/project/cache"]
  extra_hosts = ["other-host:127.0.0.1"]
  links = ["mysql_container:mysql"]
  services = ["mysql", "redis:2.8", "postgres:9"]
  allowed_images = ["ruby:*", "python:*", "php:*"]
  allowed_services = ["postgres:9.4", "postgres:latest"]
```

### Volumes in the [runners.docker] section

You can find the complete guide of Docker volume usage
[here](https://docs.docker.com/userguide/dockervolumes/).

Let's use some examples to explain how it work (assuming you have a working
runner).

#### Example 1: adding a data volume

A data volume is a specially-designated directory within one or more containers
that bypasses the Union File System. Data volumes are designed to persist data,
independent of the container's life cycle.

```bash
[runners.docker]
  host = ""
  hostname = ""
  tls_cert_path = "/Users/ayufan/.boot2docker/certs"
  image = "ruby:2.1"
  privileged = false
  disable_cache = true
  volumes = ["/path/to/volume/in/container"]
```

This will create a new volume inside the container at `/path/to/volume/in/container`.

#### Example 2: mount a host directory as a data volume

In addition to creating a volume using you can also mount a directory from your
Docker daemon's host into a container. It's useful when you want to store
builds outside the container.

```bash
[runners.docker]
  host = ""
  hostname = ""
  tls_cert_path = "/Users/ayufan/.boot2docker/certs"
  image = "ruby:2.1"
  privileged = false
  disable_cache = true
  volumes = ["/path/to/bind/from/host:/path/to/bind/in/container:rw"]
```

This will use `/path/to/bind/from/host` of the CI host inside the container at
`/path/to/bind/in/container`.

### Using a private Docker registry

_This feature requires GitLab Runner v0.6.0 or higher_

In order to use a docker image from a private registry which needs
authentication, you must first authenticate against the docker registry in
question.

If you are using our Linux packages, then `gitlab-runner` is run by the user
root (for non-root users, see the **Notes** section below).

As root run:

```bash
docker login <registry>
```

Replace `<registry>` with the Fully Qualified Domain Name of the registry and
optionally a port, for example:

```bash
docker login my.registry.tld:5000
```

The default value is `docker.io` which is the official registry Docker Inc.
provides. If you omit the registry name, `docker.io` will be implied.

After you enter the needed credentials, docker will inform you that the
credentials are saved in `/root/.docker/config.json`.

In case you are running an older Docker Engine (< 1.7.0), then the credentials
will be stored in `/root/.dockercfg`. GitLab Runner supports both locations for
backwards compatibility.

The steps performed by the Runner can be summed up to:

1. The registry name is found from the image name
1. If the value is not empty, the executor will try to look at `~/.dockercfg`
   (Using `NewAuthConfigurationsFromDockerCfg()` method in go-dockerclient)
1. If that fails for some reason, the executor will then look at
   `~/.docker/config.json` (Which should be the new default from Docker 1.7.0)
1. Finally, if an Authentication corresponding to the specified registry is
   found, subsequent Pull will make use of it

Now that the Runner is set up to authenticate against your private registry,
learn [how to configure .gitlab-ci.yml][yaml-priv-reg] in order to use that
registry.

**Notes**

If you are running `gitlab-runner` with a non-root user, you must use that user
to login to the private docker registry. This user will also need to be in the
`docker` group in order to be able to run any docker commands. To add a user to
the `docker` group use: `sudo usermod -aG user docker`.

For reference, if you want to set up your own personal registry you might want
to have a look at <https://docs.docker.com/registry/deploying/>.


### Using a Shared Docker in Docker Daemon

If you want to be able to run Docker commands in your run's, one option is to
run a [Docker daemon inside it's own container](https://hub.docker.com/_/docker/)
and link to that container from your builds.

First start up a docker daemon container on your host:

```bash
docker run --privileged --name some-docker -d docker:1.9-dind
```

Then tell your runner to use a container that contains the Docker binary and
to link the Docker daemon to it.

```toml
[runners.docker]
  # docker:1.9-git doesn't have bash installed, so this will actually not work.
  # You should build your own image that extends from this and installs at
  # least bash, and anything else you need (like Docker Compose), like
  # https://github.com/saulshanabrook/docker-compose-image
  image = "docker:1.9-git" 

  links = ["some-docker:docker"]
```

#### Mounting Volumes

if you try to mount a volume in your run, it will be empty. This is because
it will share the path from the container running the docker daemon (`some-docker`)
which doesn't have any of your files on it.

The solution is to give that container access to your files by having it share
the `/builds` directory with your run container. You should also have to tell
Docker that we are now sharing this volume, between runs, so that it will
seperarate the builds into different subdirectories.

```bash
docker run --privileged --name some-docker -v gitlab-builds:/builds -d docker:1.9-dind
```

```toml
[runners.docker]
  # docker:1.9-git doesn't have bash installed, so this will actually not work.
  # You should build your own image that extends from this and installs at
  # least bash, and anything else you need (like Docker Compose), like
  # https://github.com/saulshanabrook/docker-compose-image
  image = "docker:1.9-git" 

  links = ["some-docker:docker"]

  volumes = ["gitlab-builds:/builds"]
  
  shared_builds_dir = true

  # By default, gitlab will make volume containing your code, so it is more
  # performant to access it. We have to disable so that our shared volume will
  # be used instead.
  disable_build_volume = true 
``` 

## The [runners.parallels] section

This defines the Parallels parameters.

| Parameter | Description |
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

## The [runners.ssh] section

This defines the SSH connection parameters.

| Parameter  | Description |
| ---------- | ----------- |
| `host`     | where to connect (overridden when using `docker-ssh`) |
| `port`     | specify port, default: 22 |
| `user`     | specify user |
| `password` | specify password |
| `identity_file` | specify file path to SSH private key (id_rsa, id_dsa or id_edcsa). The file needs to be stored unencrypted |

Example:

```
[runners.ssh]
  host = "my-production-server"
  port = "22"
  user = "root"
  password = "production-server-password"
  identity_file = "
```

## Note

If you'd like to deploy to multiple servers using GitLab CI, you can create a
single script that deploys to multiple servers or you can create many scripts.
It depends on what you'd like to do.

[TOML]: https://github.com/toml-lang/toml
[Docker Engine]: https://www.docker.com/docker-engine
[yaml-priv-reg]: http://doc.gitlab.com/ce/ci/yaml/README.html#using-a-private-docker-registry
