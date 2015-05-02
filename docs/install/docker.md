## Run gitlab-ci-multi-runner in a container

### Docker image installation and configuration

Install Docker first:

```bash
curl -sSL https://get.docker.com/ | sh
```

We need to mount a data volume into our gitlab-ci-multi-runner container to
be used for configs and other resources:

```bash
docker run -d --name multi-runner --restart always \
-v /PATH/TO/DATA/FOLDER:/data \
ayufan/gitlab-ci-multi-runner:latest
```

OR you can use a data container to mount you custom data volume:

```bash
docker run -d --name multi-runner-data -v /data busybox:latest /bin/true

docker run -d --name multi-runner --restart always \
    --volumes-from multi-runner-data \
    ayufan/gitlab-ci-multi-runner:latest
```

If you plan on using Docker as the method of spawing runners, you will need to
mount your docker socket like:

```bash
docker run -d --name multi-runner --restart always \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /PATH/TO/DATA/FOLDER:/data \
  ayufan/gitlab-ci-multi-runner:latest
```

Register the runner:

```bash
docker exec -it multi-runner gitlab-ci-multi-runner register

Please enter the gitlab-ci coordinator URL (e.g. http://gitlab-ci.org:3000/ )
https://ci.gitlab.org/
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

The runner should be started already and you are ready to build your projects!

### Update

Pull the latest version:

```bash
docker pull ayufan/gitlab-ci-multi-runner:latest
```

Stop and remove the existing container:

```bash
docker stop multi-runner && docker rm multi-runner
```

Start the container as you did originally:

```bash
docker run -d --name multi-runner --restart always \
  --volumes-from multi-runner-data \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ayufan/gitlab-ci-multi-runner:latest
```

**Note**: you need to use the same method for mounting you data volume as you
    did originally (`-v /PATH/TO/DATA/FOLDER:/data` or `--volumes-from multi-runner-data`)

### Installing Trusted SSL Server Certificates

If your GitLab CI server is using self-signed SSL certificates then you should
make sure the GitLab CI server certificate is trusted by the gitlab-ci-multi-runner
container for them to be able to talk to each other.

The gitlab-ci-multi-runner image is configured to look for the trusted SSL
certificates at `/data/certs/ca.crt`, this can however be changed using the
`-e "CA_CERTIFICATES_PATH=/DIR/CERT"` configuration option.

Copy the `ca.crt` file into the `certs` directory on the data volume (or container).
The `ca.crt` file should contain the root certificates of all the servers you
want gitlab-ci-multi-runner to trust. The gitlab-ci-multi-runner container will
import the `ca.crt` file on startup so if your container is already running you
may need to restart it for the changes to take effect.
