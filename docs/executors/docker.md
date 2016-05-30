# The Docker executor

GitLab Runner can use Docker to run builds on user provided images. This is
possible with the use of **Docker** executor.

The **Docker** executor when used with GitLab CI, connects to [Docker Engine]
and runs each build in a separate and isolated container using the predefined
image that is [set up in `.gitlab-ci.yml`][yaml] and in accordance in
[`config.toml`][toml].

That way you can have a simple and reproducible build environment that can also
run on your workstation. The added benefit is that you can test all the
commands that we will explore later from your shell, rather than having to test
them on a dedicated CI server.

---

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Workflow](#workflow)
- [The `image` keyword](#the-image-keyword)
- [The `services` keyword](#the-services-keyword)
    - [How is service linked to the build](#how-is-service-linked-to-the-build)
- [Define image and services from `.gitlab-ci.yml`](#define-image-and-services-from-gitlab-ci-yml)
- [Define image and services in `config.toml`](#define-image-and-services-in-config-toml)
- [Define an image from a private Docker registry](#define-an-image-from-a-private-docker-registry)
- [Accessing the services](#accessing-the-services)
- [Configuring services](#configuring-services)
    - [PostgreSQL service example](#postgresql-service-example)
    - [MySQL service example](#mysql-service-example)
    - [The services health check](#the-services-health-check)
- [The builds and cache storage](#the-builds-and-cache-storage)
- [The persistent storage](#the-persistent-storage)
- [The privileged mode](#the-privileged-mode)
    - [Use docker-in-docker with privileged mode](#use-docker-in-docker-with-privileged-mode)
- [The ENTRYPOINT](#the-entrypoint)
- [Docker vs Docker-SSH](#docker-vs-docker-ssh)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Workflow

The Docker executor divides the build into multiple steps:

1. **Prepare**: Create and start the services.
1. **Pre-build**: Clone, restore cache and download artifacts from previous
   stages. This is run on a special Docker Image.
1. **Build**: User build. This is run on the user-provided docker image.
1. **Post-build**: Create cache, upload artifacts to GitLab. This is run on
   a special Docker Image.

The special Docker Image is based on [Alpine Linux] and contains all the tools
required to run the prepare step the build: the Git binary and the Runner
binary for supporting caching and artifacts. You can find the definition of
this special image [in the official Runner repository][special-build].

## The `image` keyword

The `image` keyword is the name of the Docker image that is present in the
local Docker Engine (list all images with `docker images`) or any image that
can be found at [Docker Hub][hub]. For more information about images and Docker
Hub please read the [Docker Fundamentals][] documentation.

In short, with `image` we refer to the docker image, which will be used to
create a container on which your build will run.

If you don't specify the namespace, Docker implies `library` which includes all
[official images](https://hub.docker.com/u/library/). That's why you'll see
many times the `library` part omitted in `.gitlab-ci.myl` and `config.toml`.
For example you can define an image like `image: ruby:2.1`, which is a shortcut
for `image: library/ruby:2.1`.

Then, for each Docker image there are tags, denoting the version of the image.
These are defined with a colon (`:`) after the image name. For example, for
Ruby you can see the supported tags at <https://hub.docker.com/_/ruby/>. If you
don't specify a tag (like `image: ruby`), `latest` is implied.

## The `services` keyword

The `services` keyword defines just another Docker image that is run during
your build and is linked to the Docker image that the `image` keyword defines.
This allows you to access the service image during build time.

The service image can run any application, but the most common use case is to
run a database container, eg. `mysql`. It's easier and faster to use an
existing image and run it as an additional container than install `mysql` every
time the project is built.

You can see some widely used services examples in the relevant documentation of
[CI services examples](https://gitlab.com/gitlab-org/gitlab-ce/tree/master/doc/ci/services/README.md).

### How is service linked to the build

To better understand how the container linking works, read
[Linking containers together](https://docs.docker.com/userguide/dockerlinks/).

To summarize, if you add `mysql` as service to your application, this image
will then be used to create a container that is linked to the build container.
According to the [workflow](#workflow) this is the first step that is performed
before running the actual builds.

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
that Runner, so even if you don't define an `image` inside `.gitlab-ci.yml`,
the one defined in `config.toml` will be used.

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
registry. Read more on [using a private Docker registry][runner-priv-reg].

## Accessing the services

Let's say that you need a Wordpress instance to test some API integration with
your application.

You can then use for example the [tutum/wordpress][] as a service image in your
`.gitlab-ci.yml`:

```yaml
services:
- tutum/wordpress:latest
```

When the build is run, `tutum/wordpress` will be started first and you will have
access to it from your build container under the hostname `tutum__wordpress`
or `tutum-wordpress`.

The GitLab Runner creates two alias hostnames for the service that you can use
alternatively. The aliases are taken from the image name following these rules:

1. Everything after `:` is stripped
2. For the first alias, the slash (`/`) is replaced with double underscores (`__`)
2. For the second alias, the slash (`/`) is replaced with a single dash (`-`)

## Configuring services

Many services accept environment variables which allow you to easily change
database names or set account names depending on the environment.

GitLab Runner 0.5.0 and up passes all YAML-defined variables to the created
service containers.

For all possible configuration variables check the documentation of each image
provided in their corresponding Docker hub page.

> **Note**:
>
All variables will be passed to all services containers. It's not designed to
distinguish which variable should go where.
>
Secure variables are only passed to the build container.

### PostgreSQL service example

See the specific documentation for
[using PostgreSQL as a service](https://gitlab.com/gitlab-org/gitlab-ce/blob/master/doc/ci/services/postgres.md).

### MySQL service example

See the specific documentation for
[using MySQL as a service](https://gitlab.com/gitlab-org/gitlab-ce/blob/master/doc/ci/services/mysql.md).

### The services health check

After the service is started, GitLab Runner waits some time for the service to
be responsive. Currently, the Docker executor tries to open a TCP connection to
the first exposed service in the service container.

You can see how it is implemented [in this Dockerfile][service-file].

## The builds and cache storage

The Docker executor by default stores all builds in
`/builds/<namespace>/<project-name>` and all caches in `/cache` (inside the
container).

You can overwrite the `/builds` and `/cache` directories by defining the
`builds_dir` and `cache_dir` options under the `[[runners]]` section in
`config.toml`. This will modify where the data are stored inside the container.

If you modify the `/cache` storage path, you also need to make sure to mark this
directory as persistent by defining it in `volumes = ["/my/cache/"]` under the
`[runners.docker]` section in `config.toml`.

Read the next section of persistent storage for more information.

## The persistent storage

The Docker executor can provide a persistent storage when running the containers.
All directories defined under `volumes =` will be persistent between builds.

The `volumes` directive supports 2 types of storage:

1. `<path>` - **the dynamic storage**. The `<path>` is persistent between subsequent
    runs of the same concurrent job for that project. The data is attached to a
    custom cache container: `runner-<short-token>-project-<id>-concurrent-<job-id>-cache-<unique-id>`.
2. `<host-path>:<path>[:<mode>]` - **the host-bound storage**. The `<path>` is
    bind to `<host-path>` on the host system. The optional `<mode>` can specify
    that this storage is read-only or read-write (default).

## The privileged mode

The Docker executor supports a number of options that allows to fine tune the
build container. One of these options is the [`privileged` mode][privileged].

### Use docker-in-docker with privileged mode

The configured `privileged` flag is passed to the build container and all
services, thus allowing to easily use the docker-in-docker approach.

First, configure your Runner (config.toml) to run in `privileged` mode:

```toml
[[runners]]
  executor = "docker"
  [runners.docker]
    privileged = true
```

Then, make your build script (`.gitlab-ci.yml`) to use Docker-in-Docker
container:

```bash
image: docker:git
services:
- docker:dind

build:
  script:
  - docker build -t my-image .
  - docker push my-image
```

## The ENTRYPOINT

The Docker executor doesn't overwrite the [`ENTRYPOINT` of a Docker image][entry].

That means that if your image defines the `ENTRYPOINT` and doesn't allow to run
scripts with `CMD`, the image will not work with the Docker executor.

With the use of `ENTRYPOINT` it is possible to create special Docker image that
would run the build script in a custom environment, or in secure mode.

You may think of creating a Docker image that uses an `ENTRYPOINT` that doesn't
execute the build script, but does execute a predefined set of commands, for
example to build the docker image from your directory. In that case, you can
run the build container in [privileged mode](#the-privileged-mode), and make
the build environment of the Runner secure.

Consider the following example:

1. Create a new Dockerfile:

    ```bash
    FROM docker:dind
    ADD / /entrypoint.sh
    ENTRYPOINT ["/bin/sh", "/entrypoint.sh"]
    ```

2. Create a bash script (`entrypoint.sh`) that will be used as the `ENTRYPOINT`:

    ```bash
    #!/bin/sh

    dind docker daemon
        --host=unix:///var/run/docker.sock \
        --host=tcp://0.0.0.0:2375 \
        --storage-driver=vf &

    docker build -t "$BUILD_IMAGE" .
    docker push "$BUILD_IMAGE"
    ```

3. Push the image to the Docker registry.

4. Run Docker executor in `privileged` mode. In `config.toml` define:

    ```toml
    [[runners]]
      executor = "docker"
      [runners.docker]
        privileged = true
    ```

5. In your project use the following `.gitlab-ci.yml`:

    ```yaml
    variables:
      BUILD_IMAGE: my.image
    build:
      image: my/docker-build:image
      script:
      - Dummy Script
    ```

This is just one of the examples. With this approach the possibilities are
limitless.

## Docker vs Docker-SSH

>**Note**:
The docker-ssh executor is deprecated and no new features will be added to it

We provide a support for a special type of Docker executor, namely Docker-SSH.
Docker-SSH uses the same logic as the Docker executor, but instead of executing
the script directly, it uses an SSH client to connect to the build container.

Docker-ssh then connects to the SSH server that is running inside the container
using its internal IP.

[Docker Fundamentals]: https://docs.docker.com/engine/understanding-docker/
[docker engine]: https://www.docker.com/products/docker-engine
[hub]: https://hub.docker.com/
[linking-containers]: https://docs.docker.com/engine/userguide/networking/default_network/dockerlinks/
[tutum/wordpress]: https://registry.hub.docker.com/u/tutum/wordpress/
[postgres-hub]: https://registry.hub.docker.com/u/library/postgres/
[mysql-hub]: https://registry.hub.docker.com/u/library/mysql/
[runner-priv-reg]: ../configuration/advanced-configuration.md#using-a-private-docker-registry
[yaml]: http://doc.gitlab.com/ce/ci/yaml/README.html
[toml]: ../commands/README.md#configuration-file
[alpine linux]: https://alpinelinux.org/
[special-build]: https://gitlab.com/gitlab-org/gitlab-ci-multi-runner/tree/master/dockerfiles/build
[service-file]: https://gitlab.com/gitlab-org/gitlab-ci-multi-runner/tree/master/dockerfiles/service
[privileged]: https://docs.docker.com/engine/reference/run/#runtime-privilege-and-linux-capabilities
[entry]: https://docs.docker.com/engine/reference/run/#entrypoint-default-command-to-execute-at-runtime
