# Install and configure GitLab Runner for auto-scaling

> The auto scale feature was introduced in GitLab Runner 1.1.0.

For an overview of the auto-scale architecture, take a look at the
[comprehensive documentation on auto-scaling](../configuration/autoscale.md).

---

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Prepare the environment](#prepare-the-environment)
- [Prepare the Docker Registry and Cache Server](#prepare-the-docker-registry-and-cache-server)
    - [Install Docker Registry](#install-docker-registry)
    - [Install the cache server](#install-the-cache-server)
- [Configure GitLab Runner](#configure-gitlab-runner)
- [Upgrading the Runner](#upgrading-the-runner)
- [Manage the Docker Machines](#manage-the-docker-machines)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Prepare the environment

In order to use the auto-scale feature, Docker and GitLab Runner must be
installed in the same machine:

1. Login to a new Linux-based machine that will serve as a bastion server and
   where Docker will spawn new machines from
1. Install GitLab Runner following the
  [GitLab Runner installation documentation][runner-installation]
1. Install Docker Machine following the
  [Docker Machine installation documentation][docker-machine-installation]

## Prepare the Docker Registry and Cache Server

To speedup the builds we advise to setup a personal Docker registry server
working in proxy mode. A cache server is also recommended.

### Install Docker Registry

>**Note:**
Read more in [Distributed registry mirroring][registry].

1. Login to a dedicated machine where Docker registry proxy will be running
2. Make sure that Docker Engine is installed on this machine
3. Create a new Docker registry:

    ```bash
    docker run -d -p 6000:5000 \
        -e REGISTRY_PROXY_REMOTEURL=https://registry-1.docker.io \
        --restart always \
        --name registry registry:2
    ```

    You can modify the port number (`6000`) to expose Docker registry on a
    different port.

4. Check the IP address of the server:

    ```bash
    hostname --ip-address
    ```

    You should preferably choose the private networking IP address. The private
    networking is usually the fastest solution for internal communication
    between machines of a single provider (DigitalOcean, AWS, Azure, etc,)
    Usually the private networking is also not accounted to your monthly
    bandwidth limit.

5. Docker registry will be accessible under `MY_REGISTRY_IP:6000`.

### Install the cache server

>**Note:**
You can use any other S3-compatible server, including [Amazon S3][S3]. Read
more in [Distributed runners caching][caching].

1. Login to a dedicated machine where the cache server will be running
1. Make sure that Docker Engine is installed on that machine
1. Start [minio], a simple S3-compatible server written in Go:

    ```bash
    docker run -it --restart always -p 9005:9000 \
            -v /.minio:/root/.minio -v /export:/export \
            --name minio \
            minio/minio:latest /export
    ```

    You can modify the port `9005` to expose the cache server on different port.

1. Check the IP address of the server:

    ```bash
    hostname --ip-address
    ```

1. Your cache server will be available at `MY_CACHE_IP:9005`
1. Read the Access and Secret Key of minio with: `sudo cat /.minio/config.json`
1. Create a bucket that will be used by the Runner: `sudo mkdir /export/runner`.
   `runner` is the name of the bucket in that case. If you choose a different
   bucket then it will be different
1. All caches will be stored in the `/export` directory

## Configure GitLab Runner

1. Register a GitLab Runner, selecting the `docker+machine` executor:

    ```bash
    sudo gitlab-ci-multi-runner register
    ```

    Example output:

    ```bash
    Please enter the gitlab-ci coordinator URL (e.g. https://gitlab.com/ci )
    https://gitlab.com/ci
    Please enter the gitlab-ci token for this runner
    xxx
    Please enter the gitlab-ci description for this runner
    my-autoscale-runner
    INFO[0034] fcf5c619 Registering runner... succeeded
    Please enter the executor: shell, docker, docker-ssh, docker+machine, docker-ssh+machine, ssh?
    docker+machine
    Please enter the Docker image (eg. ruby:2.1):
    ruby:2.1
    INFO[0037] Runner registered successfully. Feel free to start it, but if it's
    running already the config should be automatically reloaded!
    ```

1. Edit [`config.toml`][toml]. You need to fill in the options for
   `[runners.machine]` and `[runners.cache]` and configure the `MachineDriver`
   selecting your provider. Also configure `MachineOptions`, `limit` and
   `IdleCount`.

    For more information visit the dedicated page covering detailed information
    about [GitLab Runner Autoscaling][runner-autoscaling].

    Example configuration using DigitalOcean:

    ```toml
    concurrent = 20

    [[runners]]
    executor = "docker+machine"
    limit = 20
    [runners.machine]
      IdleCount = 5
      MachineDriver = "digitalocean"
      MachineName = "auto-scale-runners-%s.my.domain.com"
      MachineOptions = [
          "digitalocean-image=coreos-beta",
          "digitalocean-ssh-user=core",
          "digitalocean-access-token=MY_DIGITAL_OCEAN_TOKEN",
          "digitalocean-region=nyc2",
          "digitalocean-private-networking",
          "engine-registry-mirror=http://MY_REGISTRY_IP:6000"
        ]
    [runners.cache]
      Type = "s3"
      ServerAddress = "MY_CACHE_IP:9005"
      AccessKey = "ACCESS_KEY"
      SecretKey = "SECRET_KEY"
      BucketName = "runner"
      Insecure = true # Use Insecure only when using with Minio, without the TLS certificate enabled
    ```

1. Try to build your project. In a few seconds, if you run `docker-machine ls`
   you should see a new machine being created.

## Upgrading the Runner

1. Stop the runner:

    ```bash
    killall -SIGQUIT gitlab-runner
    ```

    Sending the [`SIGQUIT` signal][signals] will make the Runner to stop
    gracefully. It will stop accepting new jobs, and will exit as soon as the
    current builds are finished.

1. Wait until the Runner exits. You can check its status with: `gitlab-runner status`
1. You can now safely upgrade the Runner without interrupting any builds

## Manage the Docker Machines

1. Stop the Runner:

    ```bash
    killall -SIGQUIT gitlab-runner
    ```

1. Wait until the Runner exits. You can check its status with: `gitlab-runner status`
1. You can now manage (upgrade or remove) any Docker Machines with the
   [`docker-machine` command][docker-machine]

[runner-installation]: https://gitlab.com/gitlab-org/gitlab-ci-multi-runner#installation
[docker-machine-installation]: https://docs.docker.com/machine/install-machine/
[runner-autoscaling]: ../configuration/autoscale.md
[s3]: https://aws.amazon.com/s3/
[minio]: https://www.minio.io/
[caching]: ../configuration/autoscale.md#distributed-runners-caching
[registry]: ../configuration/autoscale.md#distributed-docker-registry-mirroring
[toml]: ../commands/README.md#configuration-file
[signals]: ../commands/README.md#signals
[docker-machine]: https://docs.docker.com/machine/reference/
