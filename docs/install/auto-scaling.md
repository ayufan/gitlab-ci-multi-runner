# Install and configure GitLab Runner auto-scaling

> The auto scale feature was introduced in GitLab Runner 1.1.0.

## Prepare the environment

1. Login to a new machine Linux-based machine
 
1. Install GitLab Runner following the 
  [GitLab Runner installation documentation][runner-installation]
  
1. Install the Docker Machine following the
  [Docker Machine installation documentation][docker-machine-installation]

## Prepare the Docker Registry and Cache Server

To speedup the builds we advise to setup the docker registry working in proxy mode.

### Install Docker Registry

1. Login to a dedicated machine where the docker proxy will be running.

2. Make sure that Docker Engine is installed on this machine.

3. Create a new Docker Registry:

    ```bash
    docker run -d -p 6000:5000 \
        -e REGISTRY_PROXY_REMOTEURL=https://registry-1.docker.io \
        --restart always \
        --name registry registry:2
    ```

    You can modify the `6000:` to expose Docker Registry on different port.
    
4. Check IP address of the server:

    ```bash
    hostname --ip-address
    ```
    
    You should preferably choose the private networking.
    The private networking is usually the fastest solution for internal 
    communication between machines on single provider.
    Usually the private networking is also not accounted to your monthly bandwidth limit.

5. The Docker Registry will be accessible under `MY_REGISTRY_IP:6000`.

### Install cache server

> You can use any other S3 server, including the [Amazon S3](https://aws.amazon.com/s3/).

1. Login to a dedicated machine where the cache server will be running will be running.

2. Make sure that Docker Engine is installed on that machine.

3. Start a [minio](https://www.minio.io/) a simple S3-compatible server written in Go:

    ```bash
    docker run -it --restart always -p 9005:9000 \
            -v /.minio:/.minio -v /export:/export \
            --name minio \
            minio/minio:latest server /export
    ```

    You can modify the `9005:` to expose cache server on different port.
    
4. Check IP address of the server:

    ```bash
    hostname --ip-address
    ```
    
5. Your cache server will be available at `MY_CACHE_IP:9005`.

6. Read the Access and Secret Key from: `sudo cat /.minio/config.json`

7. Create a backet that will be used by runner: `sudo mkdir /export/runner`.

8. All caches will be stored in `/export` directory.

## Configure GitLab Runner

1. Register GitLab Runner, selecting the `docker+machine` executor:

    ```bash
    sudo gitlab-ci-multi-runner register
    
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

2. Edit the `/etc/gitlab-runner/config.toml`. You need to fill the `[runners.machine]` and `[runners.cache]`:

    You need to configure the `MachineDriver` selecting your provider, configure `MachineOptions`, `limit` and `IdleCount`.
    
    For more information visit the dedicated page covering detailed information about [GitLab Runner Autoscaling][runner-autoscaling].

    **Example configuration:**

    ```
    concurrent = 20

    [[runners]]
    executor = "docker+machine"
    limit = 20
    ...
    [runners.machine]
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
        IdleCount = 5
    [runners.cache]
        Type = "s3"
        ServerAddress = "MY_CACHE_IP:9005"
        AccessKey = "ACCESS_KEY"
        SecretKey = "SECRET_KEY"
        BucketName = "runner"
        Insecure = true # Use Insecure only when using with Minio, without the TLS certificate enabled
    ```

3. Try to build your project. In a few seconds when you do `docker-machine ls` you should see a new machine being created.

## Upgrade the Runner

1. Stop the runner: `killall -SIGQUIT gitlab-runner`. Sending the SIGQUIT will make the runner to stop gracefully.
Runner will stop accepting new jobs, and it will exit as soon as it finished current builds.

1. Wait till the runner exits. You can check the status with: `gitlab-runner status`.

1. You can now safely upgrade runner without interrupting builds.

## Manage the Machines

1. Stop the runner: `killall -SIGQUIT gitlab-runner`.

1. Wait till the runner exits. You can check the status with: `gitlab-runner status`.

1. You can now manage (upgrade or remove) the Docker Machines with `docker-machine`.

[runner-installation]: https://gitlab.com/gitlab-org/gitlab-ci-multi-runner#installation
[docker-machine-installation]: https://docs.docker.com/machine/install-machine/
[runner-autoscaling]: https://gitlab.com/gitlab-org/gitlab-ci-multi-runner/tree/master/docs/configuration/configuration/autoscaling.md
