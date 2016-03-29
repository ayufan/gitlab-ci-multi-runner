## Run gitlab-runner in a container

### Docker image installation and configuration

Install Docker first:

```bash
curl -sSL https://get.docker.com/ | sh
```

We need to mount a config volume into our gitlab-runner container to
be used for configs and other resources:

```bash
docker run -d --name gitlab-runner --restart always \
  -v /srv/gitlab-runner/config:/etc/gitlab-runner \
  gitlab/gitlab-runner:latest
```

OR you can use a config container to mount your custom data volume:

```bash
docker run -d --name gitlab-runner-config \
    -v /etc/gitlab-runner \
    busybox:latest \
    /bin/true

docker run -d --name gitlab-runner --restart always \
    --volumes-from gitlab-runner-config \
    gitlab/gitlab-runner:latest
```

If you plan on using Docker as the method of spawning runners, you will need to
mount your docker socket like this:

```bash
docker run -d --name gitlab-runner --restart always \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /srv/gitlab-runner/config:/etc/gitlab-runner \
  gitlab/gitlab-runner:latest
```

Register the runner:

```bash
docker exec -it gitlab-runner gitlab-runner register

Please enter the gitlab-ci coordinator URL (e.g. https://gitlab.com/ci )
https://gitlab.com/ci
Please enter the gitlab-ci token for this runner
xxx
Please enter the gitlab-ci description for this runner
my-runner
INFO[0034] fcf5c619 Registering runner... succeeded
Please enter the executor: shell, docker, docker-ssh, ssh?
docker
Please enter the Docker image (eg. ruby:2.1):
ruby:2.1
INFO[0037] Runner registered successfully. Feel free to start it, but if it's
running already the config should be automatically reloaded!
```

The runner should is started already and you are ready to build your projects!

Make sure that you read the [FAQ](../faq/README.md) section which describes
some of the most common problems with GitLab Runner.

### Update

Pull the latest version:

```bash
docker pull gitlab/gitlab-runner:latest
```

Stop and remove the existing container:

```bash
docker stop gitlab-runner && docker rm gitlab-runner
```

Start the container as you did originally:

```bash
docker run -d --name gitlab-runner --restart always \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /srv/gitlab-runner/config:/etc/gitlab-runner \
  gitlab/gitlab-runner:latest
```

**Note**: you need to use the same method for mounting you data volume as you
    did originally (`-v /srv/gitlab-runner/config:/etc/gitlab-runner` or `--volumes-from gitlab-runner`)

### Installing Trusted SSL Server Certificates

If your GitLab CI server is using self-signed SSL certificates then you should
make sure the GitLab CI server certificate is trusted by the gitlab-ci-multi-runner
container for them to be able to talk to each other.

The `gitlab/gitlab-runner` image is configured to look for the trusted SSL
certificates at `/etc/gitlab-runner/certs/ca.crt`, this can however be changed using the
`-e "CA_CERTIFICATES_PATH=/DIR/CERT"` configuration option.

Copy the `ca.crt` file into the `certs` directory on the data volume (or container).
The `ca.crt` file should contain the root certificates of all the servers you
want gitlab-ci-multi-runner to trust. The gitlab-ci-multi-runner container will
import the `ca.crt` file on startup so if your container is already running you
may need to restart it for the changes to take effect.

### Alpine Linux

You can also use alternative [Alpine Linux](https://www.alpinelinux.org/) based image with much smaller footprint:
```
gitlab/gitlab-runner    latest              3e8077e209f5        13 hours ago        304.3 MB
gitlab/gitlab-runner    alpine              7c431ac8f30f        13 hours ago        25.98 MB
```

**Alpine Linux image is designed to use only Docker as the method of spawning runners.**

The original `gitlab/gitlab-runner:latest` is based on Ubuntu 14.04 LTS.

### SELinux

Some distributions (CentOS, RedHat, Fedora) use SELinux by default to enhance the security of the underlying system.

The special care must be taken when dealing with such configuration.

1. If you want to use Docker executor to run builds in containers you need to access the `/var/run/docker.sock`.
However, if you have a SELinux in enforcing mode, you will see the `Permission denied` when accessing the `/var/run/docker.sock`.
Install the `selinux-dockersock` and to resolve the issue: https://github.com/dpw/selinux-dockersock.

1. Make sure that persistent directory is created on host: `mkdir -p /srv/gitlab-runner/config`.

1. Run docker with `:Z` on volumes:

```bash
    docker run -d --name gitlab-runner --restart always \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -v /srv/gitlab-runner/config:/etc/gitlab-runner:Z \
      gitlab/gitlab-runner:latest
```

More information about the cause and resolution can be found here:
http://www.projectatomic.io/blog/2015/06/using-volumes-with-docker-can-cause-problems-with-selinux/
