# Docker

The GitLab Runner can use Docker to run builds on user provided images.
This is possible with the use of **Docker** executor.
 
The **Docker** executor connects to any [Docker Engine](https://www.docker.com/products/docker-engine).
Docker, when used with GitLab CI, runs each build in a separate and isolated container using the predefined image
that is set up in .gitlab-ci.yml.

This makes it easier to have a simple and reproducible build environment that can also run on your workstation.
The added benefit is that you can test all the commands that we will explore later from your shell,
rather than having to test them on a dedicated CI server.

## The workflow

The **Docker** executor divides the build into multiple steps:
1. **Prepare**: Create and start the services.
1. **Pre-build**: Clone, restore cache and download artifacts from previous stages. This is run on special Docker Image.
1. **Build**: User build. This is run on user-provided docker image
1. **Post-build**: Create cache, upload artifacts to GitLab. This is run on special Docker Image.

The special Docker Image is based on Alpine Linux and contains all tools required to run the prepare the build:
the git, the runner binary for supporting caching and artifacts.
You can find the definition of this special image here: https://gitlab.com/gitlab-org/gitlab-ci-multi-runner/tree/master/dockerfiles/build.

## The image

The `image` keyword is the name of the docker image that is present in the
local Docker Engine (list all images with `docker images`) or any image that
can be found at [Docker Hub][hub]. For more information about images and Docker
Hub please read the [Docker Fundamentals][] documentation.

In short, with `image` we refer to the docker image, which will be used to
create a container on which your build will run.

## The services

The `services` keyword defines just another docker image that is run during
your build and is linked to the docker image that the `image` keyword defines.
This allows you to access the service image during build time.

The service image can run any application, but the most common use case is to
run a database container, eg. `mysql`. It's easier and faster to use an
existing image and run it as an additional container than install `mysql` every
time the project is built.

You can see some widely used services examples in the relevant documentation of
[CI services examples](../services/README.md).

### How is service linked to the build

To better understand how the container linking works, read
[Linking containers together](https://docs.docker.com/userguide/dockerlinks/).

To summarize, if you add `mysql` as service to your application, the image will
then be used to create a container that is linked to the build container.

The service container for MySQL will be accessible under the hostname `mysql`.
So, in order to access your database service you have to connect to the host
named `mysql` instead of a socket or `localhost`.

## Define image and services from `.gitlab-ci.yml`

You can simply define an image that will be used for all jobs and a list of
services that you want to use during build time.

```yaml
image: ruby:2.2

services:
  - postgres:9.3

before_script:
  - bundle install

test:
  script:
  - bundle exec rake spec
```

It is also possible to define different images and services per job:

```yaml
before_script:
  - bundle install

test:2.1:
  image: ruby:2.1
  services:
  - postgres:9.3
  script:
  - bundle exec rake spec

test:2.2:
  image: ruby:2.2
  services:
  - postgres:9.4
  script:
  - bundle exec rake spec
```

## Define image and services in `config.toml`

Look for the `[runners.docker]` section:

```
[runners.docker]
  image = "ruby:2.1"
  services = ["mysql:latest", "postgres:latest"]
```

The image and services defined this way will be added to all builds run by
that runner.

## Define an image from a private Docker registry

Starting with GitLab Runner 0.6.0, you are able to define images located to
private registries that could also require authentication.

All you have to do is be explicit on the image definition in `.gitlab-ci.yml`.

```yaml
image: my.registry.tld:5000/namepace/image:tag
```

In the example above, GitLab Runner will look at `my.registry.tld:5000` for the
image `namespace/image:tag`.

If the repository is private you need to authenticate your GitLab Runner in the
registry. Learn how to do that on
[GitLab Runner's documentation][runner-priv-reg].

## Accessing the services

Let's say that you need a Wordpress instance to test some API integration with
your application.

You can then use for example the [tutum/wordpress][] image in your
`.gitlab-ci.yml`:

```yaml
services:
- tutum/wordpress:latest
```

When the build is run, `tutum/wordpress` will be started and you will have
access to it from your build container under the hostname `tutum__wordpress`
or `tutum-wordpress`.

The GitLab Runner creates two alias hostnames for the service.
The alias is made from the image name following these
rules:

1. Everything after `:` is stripped
2. For first alias replace all slash (`/`) is replaced with double underscores (`__`)
2. For second alias replace all slash (`/`) is replaced with double underscores (`__`)

## Configuring services

Many services accept environment variables which allow you to easily change
database names or set account names depending on the environment.

GitLab Runner 0.5.0 and up passes all YAML-defined variables to the created
service containers.

For all possible configuration variables check the documentation of each image
provided in their corresponding Docker hub page.

*Note: All variables will be passed to all services containers. It's not
designed to distinguish which variable should go where.*

*Note: Secure variables are only passed to build container.*

### PostgreSQL service example

See the specific documentation for
[using PostgreSQL as a service](../services/postgres.md).

### MySQL service example

See the specific documentation for
[using MySQL as a service](../services/mysql.md).

### The services health check

After the service is started the GitLab Runner waits some time for the service to be responsive.
Currently the docker executor tries to open TCP connection to first exposed service in service container.

You can see how it is implemented here: https://gitlab.com/gitlab-org/gitlab-ci-multi-runner/tree/master/dockerfiles/service.

## The builds and cache storage

Docker executor by default stores all builds in `/builds/<group-name>/<project-name>`
and all caches in `/cache`.

You can overwrite the `/builds` and `/cache` by defining the `builds_dir` and `cache_dir`
for `[[runners]]` of config.toml.
This will modify where the data are stored in perspective of the container.

If you modyfi the `/cache` storage you also need to make sure to mark this directory as persistent
by defining it in `volumes = ["/my/cache/"]` in `[runners.docker]`.

## The persistent storage

Docker executor can provide a persistent storage to build containers.
All directories defined `volumes =` will be persistent between builds.

The `volumes` directive support 2 types of storage:
1. `<path>` - the dynamic storage, the `<path>` is persistent between subsequent runs of the same concurrent job for that project.
The data is attached to custom cache container `runner-<short-token>-project-<id>-concurrent-<job-id>-cache-<unique-id>`.

2. `<host-path>:<path>[:<mode>]` - the host-bound storage, the `<path>` is binded to `<host-path>` on host system.
The optional `<mode>` can specify that this storage is read-only or read-write (default).

## The privileged mode

Docker executor supports a number of options that allows to fine tune the build container.
One of this option is the [`privileged`](https://docs.docker.com/engine/reference/run/#runtime-privilege-and-linux-capabilities) mode.

The configured privileged flag is passed to the build container and all services, thus allowing to easily use the docker-in-docker approach.

### Use docker-in-docker with privileged mode

Configure your Runner (config.toml) to run in `privileged` mode:

    ```
    [[runners]]
    executor = "docker"
    ...
    [runners.docker]
    privileged = true
    ```

Make your build script (.gitlab-ci.yml) to use Docker-in-Docker container:

    image: docker:git
    services:
    - docker:dind
    
    build:
      script:
      - docker build -t my-image .
      - docker push my-image

## The entrypoint

**Docker** executor doesn't overwrite the **ENTRYPOINT** of Docker Image.

So if your image defines the **ENTRYPOINT** and doesn't allow to run scripts with **CMD**
the image will not work with Docker executor.

With the use of **ENTRYPOINT** it is possible to create special docker image that would run
the build script in custom environment, or in secure mode.

You could think of creating docker image that uses **ENTRYPOINT** that doesn't execute the build script,
but does execute some predefined set of commands, for example to build the docker image from your directory.
In that case you can run the build container in privileged mode, and make the build environment and runner secure.

### Example

1. Create a new Dockerfile:

    ```
    FROM docker:dind
    ADD / /my.entrypoint.sh
    ENTRYPOINT ["/bin/sh", "/my.entrypoint.sh"]
    ```

2. Create a `/my.entrypoint.sh`:

    ```
    #!/bin/sh
    
    dind docker daemon
        --host=unix:///var/run/docker.sock \
        --host=tcp://0.0.0.0:2375 \
        --storage-driver=vf &

    docker build -t "$BUILD_IMAGE" .
    docker push "$BUILD_IMAGE"
    ```

3. Push the image to registry.

4. Run Docker executor in `privileged` mode.

5. In your project use the following `.gitlab-ci.yml`:

    ```
    variables:
      BUILD_IMAGE: my.image
    build:
      image: my/docker-build:image
      script:
      - Dummy Script
    ``` 

This is just one of the examples. With this approach the possibilities are limitless.

## Docker vs Docker-SSH

*Note: the docker-ssh executor is deprecated and no new features will be added to it*

We provide a support for special type of Docker executor, the Docker-SSH.
Docker-SSH uses the same logic as Docker executor,
but instead of executing the script directly it uses SSH client to connect to
build container.

The docker-ssh connects then to SSH server running on internal IP of the container.

[Docker Fundamentals]: https://docs.docker.com/engine/understanding-docker/
[hub]: https://hub.docker.com/
[linking-containers]: https://docs.docker.com/engine/userguide/networking/default_network/dockerlinks/
[tutum/wordpress]: https://registry.hub.docker.com/u/tutum/wordpress/
[postgres-hub]: https://registry.hub.docker.com/u/library/postgres/
[mysql-hub]: https://registry.hub.docker.com/u/library/mysql/
[runner-priv-reg]: https://gitlab.com/gitlab-org/gitlab-ci-multi-runner/blob/master/docs/configuration/advanced-configuration.md#using-a-private-docker-registry