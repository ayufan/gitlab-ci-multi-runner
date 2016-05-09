# Advanced configuration

GitLab Runner configuration uses the [TOML][] format.

The file to be edited can be found in:

1. `/etc/gitlab-runner/config.toml` on \*nix systems when gitlab-runner is
   executed as root (**this is also path for service configuration**)
1. `~/.gitlab-runner/config.toml` on \*nix systems when gitlab-runner is
   executed as non-root
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
| `cache_dir`         | directory where build caches will be stored in context of selected executor (Locally, Docker, SSH). If the `docker` executor is used, this directory needs to be included in its `volumes` parameter. |
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
| `docker-ssh`  | run build using Docker container, but connect to it with SSH - this requires the presence of `[runners.docker]` , `[runners.ssh]` and [Docker Engine][] installed on the system that the Runner runs. **Note: This will run the docker container on the local machine, it just changes how the commands are run inside that container. If you want to run docker commands on an external machine, then you should change the `host` parameter in the `runners.docker` section.**|
| `ssh`         | run build remotely with SSH - this requires the presence of `[runners.ssh]` |
| `parallels`   | run build using Parallels VM, but connect to it with SSH - this requires the presence of `[runners.parallels]` and `[runners.ssh]` |
| `virtualbox`  | run build using VirtualBox VM, but connect to it with SSH - this requires the presence of `[runners.virtualbox]` and `[runners.ssh]` |
| `docker+machine` | like `docker`, but uses [auto-scaled docker machines](autoscale.md) - this requires the presence of `[runners.docker]` and `[runners.machine]` |
| `docker-ssh+machine` | like `docker-ssh`, but uses [auto-scaled docker machines](autoscale.md) - this requires the presence of `[runners.docker]` and `[runners.machine]` |

## The SHELLS

There are a couple of available shells that can be run on different platforms.

| Shell | Description |
| ----- | ----------- |
| `bash`        | generate Bash (Bourne-shell) script. All commands executed in Bash context (default for all Unix systems) |
| `sh`          | generate Sh (Bourne-shell) script. All commands executed in Sh context (fallback for `bash` for all Unix systems) |
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
| `cap_add`                   | add additional Linux capabilities to the container |
| `cap_drop`                  | drop additional Linux capabilities from the container |
| `devices`                   | share additional host devices with the container |
| `disable_cache`             | disable automatic |
| `wait_for_services_timeout` | specify how long to wait for docker services, set to 0 to disable, default: 30 |
| `cache_dir`                 | specify where Docker caches should be stored (this can be absolute or relative to current working directory) |
| `volumes`                   | specify additional volumes that should be mounted (same syntax as Docker -v option) |
| `extra_hosts`               | specify hosts that should be defined in container environment |
| `links`                     | specify containers which should be linked with building container |
| `services`                  | specify additional services that should be run with build. Please visit [Docker Registry](https://registry.hub.docker.com/) for list of available applications. Each service will be run in separate container and linked to the build. |
| `allowed_images`            | specify wildcard list of images that can be specified in .gitlab-ci.yml |
| `allowed_services`          | specify wildcard list of services that can be specified in .gitlab-ci.yml |
| `pull_policy`               | specify the image pull policy: never, if-not-present or always (default) |

Example:

```bash
[runners.docker]
  host = ""
  hostname = ""
  tls_cert_path = "/Users/ayufan/.boot2docker/certs"
  image = "ruby:2.1"
  privileged = false
  cap_add = ["NET_ADMIN"]
  cap_drop = ["DAC_OVERRIDE"]
  devices = ["/dev/net/tun"]
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

## The [runners.virtualbox] section

This defines the VirtualBox parameters. This executor relies on
`vboxmanage` as executable to control VirtualBox machines so you have to adjust
your `PATH` environment variable on Windows hosts:
`PATH=%PATH%;C:\Program Files\Oracle\VirtualBox`.

| Parameter | Explanation |
| --------- | ----------- |
| `base_name`         | name of VirtualBox VM which will be cloned |
| `disable_snapshots` | if disabled the VMs will be destroyed after build |

Example:

```bash
[runners.virtualbox]
  base_name = "my-virtualbox-image"
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
  identity_file = ""
```

## The [runners.machine] section

>**Note:**
Added in GitLab Runner v1.1.0.

This defines the Docker Machine based autoscaling feature. More details can be
found in the separate [runners autoscale documentation](autoscale.md).

| Parameter        | Description |
|------------------|-------------|
| `IdleCount`      | Number of machines, that need to be created and waiting in _Idle_ state. |
| `IdleTime`       | Time (in seconds) for machine to be in _Idle_ state before it is removed. |
| `MaxBuilds`      | Builds count after which machine will be removed. |
| `MachineName`    | Name of the machine. It **must** contain `%s`, which will be replaced with a unique machine identifier. |
| `MachineDriver`  | Docker Machine `driver` to use. More details can be found in the [Docker Machine configuration section](autoscale.md#what-are-the-supported-cloud-providers). |
| `MachineOptions` | Docker Machine options. More details can be found in the [Docker Machine configuration section](autoscale.md#what-are-the-supported-cloud-providers). |

Example:

```bash
[runners.machine]
  IdleCount = 5
  IdleTime = 600
  MaxBuilds = 100
  MachineName = "auto-scale-%s"
  MachineDriver = "digitalocean"
  MachineOptions = [
      "digitalocean-image=coreos-beta",
      "digitalocean-ssh-user=core",
      "digitalocean-access-token=DO_ACCESS_TOKEN",
      "digitalocean-region=nyc2",
      "digitalocean-size=4gb",
      "digitalocean-private-networking",
      "engine-registry-mirror=http://10.11.12.13:12345"
  ]
```

## The [runners.cache] section

>**Note:**
Added in GitLab Runner v1.1.0.

This defines the distributed cache feature. More details can be found
in the [runners autoscale documentation](autoscale.md#distributed-runners-caching).

| Parameter        | Type             | Description |
|------------------|------------------|-------------|
| `Type`           | string           | As of now, only S3-compatible services are supported, so only `s3` can be used. |
| `ServerAddress`  | string           | A `host:port` to the used S3-compatible server. |
| `AccessKey`      | string           | The access key specified for your S3 instance. |
| `SecretKey`      | string           | The secret key specified for your S3 instance. |
| `BucketName`     | string           | Name of the bucket where cache will be stored. |
| `Insecure`       | boolean          | Set to `true` if the S3 service is available by `HTTP`. Is set to `false` by default. |

Example:

```bash
[runners.cache]
  Type = "s3"
  ServerAddress = "s3.amazonaws.com"
  AccessKey = "AMAZON_S3_ACCESS_KEY"
  SecretKey = "AMAZON_S3_SECRET_KEY"
  BucketName = "runners"
  Insecure = false
```

> **Note:** For Amazon's S3 service the `ServerAddress` should always be `s3.amazonaws.com`. Minio S3 client will
> get bucket metadata and modify the URL to point to the valid region (eg. `s3-eu-west-1.amazonaws.com`) itself.

## Note

If you'd like to deploy to multiple servers using GitLab CI, you can create a
single script that deploys to multiple servers or you can create many scripts.
It depends on what you'd like to do.

[TOML]: https://github.com/toml-lang/toml
[Docker Engine]: https://www.docker.com/docker-engine
[yaml-priv-reg]: http://doc.gitlab.com/ce/ci/yaml/README.html#using-a-private-docker-registry
