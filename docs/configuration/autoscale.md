# Runners autoscale configuration

> The autoscale feature was introduced in GitLab Runner 1.1.0.

---

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Overview](#overview)
- [System requirements](#system-requirements)
- [Runner configuration](#runner-configuration)
    - [Runner global options](#runner-global-options)
    - [`[[runners]]` options](#runners-options)
    - [`[runners.machine]` options](#runners-machine-options)
    - [`[runners.cache]` options](#runners-cache-options)
    - [Additional configuration information](#additional-configuration-information)
- [Autoscaling algorithm and parameters](#autoscaling-algorithm-and-parameters)
- [How `current`, `limit` and `IdleCount` generate the upper limit of running machines](#how-current-limit-and-idlecount-generate-the-upper-limit-of-running-machines)
- [Distributed runners caching](#distributed-runners-caching)
- [Distributed Docker registry mirroring](#distributed-docker-registry-mirroring)
- [A complete example of `config.toml`](#a-complete-example-of-config-toml)
- [What are the supported cloud providers](#what-are-the-supported-cloud-providers)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

Autoscale provides the ability to utilize resources in a more elastic and
dynamic way.

When this feature is enabled and configured properly, builds are executed on
machines created _on demand_. Those machines, after the build is finished, can
wait to run the next builds or can be removed after the configured `IdleTime`.
In case of many cloud providers this helps to utilize the cost of already used
instances.

Thanks to runners being able to autoscale, your infrastructure contains only as
much build instances as necessary at anytime. If you configure the Runner to
only use autoscale, the system on which the Runner is installed acts as a
bastion for all the machines it creates.

Below, you can see a real life example of the runners autoscale feature, tested
on GitLab.com for the [GitLab Community Edition][ce] project:

![Real life example of autoscaling](img/autoscale-example.png)

Each machine on the chart is an independent cloud instance, running build jobs
inside of Docker containers.

[ce]: https://gitlab.com/gitlab-org/gitlab-ce

## System requirements

To use the autoscale feature, the system which will host the Runner must have:

- GitLab Runner executable - installation guide can be found in
  [GitLab Runner Documentation][runner-installation]
- Docker Machine executable - installation guide can be found in
  [Docker Machine documentation][docker-machine-installation]

If you need to use any virtualization/cloud providers that aren't handled by
Docker's Machine internal drivers, the appropriate driver plugin must be
installed. The Docker Machine driver plugin installation and configuration is
out of the scope of this documentation. For more details please read the
[Docker Machine documentation][docker-machine-docs].

## Runner configuration

In this section we will describe only the significant parameters from the
autoscale feature point of view. For more configurations details please read
the [GitLab Runner - Installation][runner-installation]
and [GitLab Runner - Advanced Configuration][runner-configuration].

### Runner global options

| Parameter    | Value   | Description |
|--------------|---------|-------------|
| `concurrent` | integer | Limits how many jobs globally can be run concurrently. This is the most upper limit of number of jobs using _all_ defined runners, local and autoscale. Together with `limit` (from [`[[runners]]` section](#runners-options)) and `IdleCount` (from [`[runners.machine]` section](advanced-configuration.md#the-runnersmachine-section)) it affects the upper limit of created machines. |

### `[[runners]]` options

| Parameter  | Value            | Description |
|------------|------------------|-------------|
| `executor` | string           | To use the autoscale feature, `executor` must be set to `docker+machine` or `docker-ssh+machine`. |
| `limit`    | integer          | Limits how many jobs can be handled concurrently by this specific token. 0 simply means don't limit. For autoscale it's the upper limit of machines created by this provider (in conjuction with `concurrent` and `IdleCount`). |

### `[runners.machine]` options

Configuration parameters details can be found
in [GitLab Runner - Advanced Configuration - The runners.machine section](advanced-configuration.md#the-runnersmachine-section).

### `[runners.cache]` options

Configuration parameters details can be found
in [GitLab Runner - Advanced Configuration - The runners.cache section](advanced-configuration.md#the-runnerscache-section)

### Additional configuration information

There is also a special mode, when you set `IdleCount = 0`. In this mode,
machines are **always** created **on-demand** before each build (if there is no
available machine in _Idle_ state). After the build is finished, the autoscaling
algorithm works
[the same as it is described below](#autoscaling-algorithm-and-parameters).
The machine is waiting for the next builds, and if no one is executed, after
the `IdleTime` period, the machine is removed. If there are no builds, there
are no machines in _Idle_ state.

## Autoscaling algorithm and parameters

The autoscaling algorithm is based on three main parameters: `IdleCount`,
`IdleTime` and `limit`.

We say that each machine that does not run a build is in _Idle_ state. When
GitLab Runner is in autoscale mode, it monitors all machines and ensures that
there is always an `IdleCount` of machines in _Idle_ state.

At the same time, GitLab Runner is checking the duration of the _Idle_ state of
each machine. If the time exceeds the `IdleTime` value, the machine is
automatically removed.

---

**Example:**
Let's suppose, that we have configured GitLab Runner with the following
autoscale parameters:

```bash
[[runners]]
  limit = 10
  (...)
  executor = "docker+machine"
  [runners.machine]
    IdleCount = 2
    IdleTime = 1800
    (...)
```

At the beginning, when no builds are queued, GitLab Runner starts two machines
(`IdleCount = 2`), and sets them in _Idle_ state. If there is 30 minutes
(`IdleTime = 1800`) of inactivity (since last project finished building), both
machines will be removed. As of this moment we have **zero** machines in _Idle_
state, so GitLab Runner starts 2 new machines to satisfy `IdleCount` which is
set to 2.

Now, let's assume that 5 builds are queued in GitLab CI. The first 2 builds are
sent to the _Idle_ machines. GitLab Runner notices that the number of _Idle_
machines is less than `IdleCount` (`0 < 2`), so it starts 2 new machines. Then,
the next 2 builds from the queue are sent to those newly created machines.
Again, the number of _Idle_ machines is less than `IdleCount`, so GitLab Runner
starts 2 new machines and the last queued build is sent to one of the _Idle_
machines.

We now have 1 _Idle_ machine, so GitLab Runner starts another 1 new machine to
satisfy `IdleCount`. Because there are no new builds in queue, those two
machines stay in _Idle_ state and GitLab Runner is satisfied.

---

**This is what happend:**
We had 2 machines, waiting in _Idle_ state for new builds. After the 5 builds
where queued, new machines were created, so in total we had 7 machines. Five of
them were running builds, and 2 were in _Idle_ state, waiting for the next
builds.

The algorithm will still work in the same way; GitLab Runner will create a new
_Idle_ machine for each machine used for the build execution until `IdleCount`
is satisfied. Those machines will be created up to the number defined by
`limit` parameter. If GitLab Runner notices that there is a `limit` number of
total created machines, it will stop autoscaling, and new builds will need to
wait in the build queue until machines start returning to _Idle_ state.

In the above example we will always have two idle machines. The `IdleTime`
applies only when we are over the `IdleCount`, then we try to reduce the number
of machines to `IdleCount`.

---

**Scaling down:**
After the build is finished, the machine is set to _Idle_ state and is waiting
for the next builds to be executed. Let's suppose that we have no new builds in
the queue. After the time designated by `IdleTime` passes, the _Idle_ machines
will be removed. In our example, after 30 minutes, all machines will be removed
(each machine after 30 minutes from when last build execution ended) and GitLab
Runner will start to keep an `IdleCount` of _Idle_ machines running, just like
at the beginning of the example.

---

So, to sum up:

1. We start the Runner
2. Runner creates 2 idle machines
3. Runner picks one build
4. Runner creates one more machine to fulfill the strong requirement of always
   having the two idle machines
5. Build finishes, we have 3 idle machines
6. When one of the three idle machines goes over `IdleTime` from the time when
   last time it picked the build it will be removed
7. The Runner will always have at least 2 idle machines waiting for fast
   picking of the builds

Below you can see a comparison chart of builds statuses and machines statuses
in time:

![Autoscale state chart](img/autoscale-state-chart.png)

## How `current`, `limit` and `IdleCount` generate the upper limit of running machines

There doesn't exist a magic equation that will tell you what to set `limit` or
`concurrent` to. Act according to your needs. Having `IdleCount` of _Idle_
machines is a speedup feature. You don't need to wait 10s/20s/30s for the
instance to be created. But as a user, you'd want all your machines (for which
you need to pay) to be running builds, not stay in _Idle_ state. So you should
have `concurrent` and `limit` set to values that will run the maximum count of
machines you are willing to pay for. As for `IdleCount`, it should be set to a
value that will generate a minimum amount of _not used_ machines when the build
queue is empty.

Let's assume the following example:

```bash
concurrent=20

[[runners]]
  limit = 40
  [runners.machine]
    IdleCount = 10
```

In the above scenario the total amount of machines we could have is 30. The
`limit` of total machines (building and idle) can be 40. We can have 10 idle
machines but the `concurrent` builds are 20. So in total we can have 20
concurrent machines running builds and 10 idle, summing up to 30.

But what happens if the `limit` is less than the total amount of machines that
could be created? The example below explains that case:

```bash
concurrent=20

[[runners]]
  limit = 25
  [runners.machine]
    IdleCount = 10
```

In this example we will have at most 20 concurrent builds, and at most 25
machines created. In the worst case scenario regarding idle machines, we will
not be able to have 10 idle machines, but only 5, because the `limit` is 25.

## Distributed runners caching

To speed up your builds, GitLab Runner provides a [cache mechanism][cache]
where selected directories and/or files are saved and shared between subsequent
builds.

This is working fine when builds are run on the same host, but when you start
using the Runners autoscale feature, most of your builds will be running on a
new (or almost new) host, which will execute each build in a new Docker
container. In that case, you will not be able to take advantage of the cache
feature.

To overcome this issue, together with the autoscale feature, the distributed
Runners cache feature was introduced.

It uses any S3-compatible server to share the cache between used Docker hosts.
When restoring and archiving the cache, GitLab Runner will query the S3 server
and will download or upload the archive.

To enable distributed caching, you have to define it in `config.toml` using the
[`[runners.cache]` directive][runners-cache]:

```bash
[[runners]]
  limit = 10
  executor = "docker+machine"
  [runners.cache]
    Type = "s3"
    ServerAddress = "s3.example.com"
    AccessKey = "access-key"
    SecretKey = "secret-key"
    BucketName = "runner"
    Insecure = false
```

Read how to [install your own caching server][caching].

## Distributed Docker registry mirroring

To speed up builds executed inside of Docker containers, you can use the [Docker
registry mirroring service][registry]. This will provide a proxy between your
Docker machines and all used registries. Images will be downloaded once by the
registry mirror. On each new host, or on an existing host where the image is
not available, it will be downloaded from the configured registry mirror.

Provided that the mirror will exist in your Docker machines LAN, the image
downloading step should be much faster on each host.

To configure the Docker registry mirroring, you have to add `MachineOptions` to
the configuration in `config.toml`:

```bash
[[runners]]
  limit = 10
  executor = "docker+machine"
  [runners.machine]
    (...)
    MachineOptions = [
      (...)
      "engine-registry-mirror=http://10.11.12.13:12345"
    ]
```

Where `10.11.12.13:12345` is the IP address and port where your registry mirror
is listening for connections from the Docker service. It must be accessible for
each host created by Docker Machine.

Read how to [install your own Docker registry server][registry-server].

## A complete example of `config.toml`

The `config.toml` below uses the `digitalocean` Docker Machine driver:

```bash
concurrent = 50   # All registered Runners can run up to 50 concurrent builds

[[runners]]
  url = "https://gitlab.com/ci"
  token = "RUNNER_TOKEN"
  name = "autoscale-runner"
  executor = "docker+machine"       # This Runner is using the 'docker+machine' executor
  limit = 10                        # This Runner can execute up to 10 builds (created machines)
  [runners.docker]
    image = "ruby:2.1"              # The default image used for builds is 'ruby:2.1'
  [runners.machine]
    IdleCount = 5                   # There must be 5 machines in Idle state
    IdleTime = 600                  # Each machine can be in Idle state up to 600 seconds (after this it will be removed)
    MaxBuilds = 100                 # Each machine can handle up to 100 builds in a row (after this it will be removed)
    MachineName = "auto-scale-%s"   # Each machine will have a unique name ('%s' is required)
    MachineDriver = "digitalocean"  # Docker Machine is using the 'digitalocean' driver
    MachineOptions = [
        "digitalocean-image=coreos-beta",
        "digitalocean-ssh-user=core",
        "digitalocean-access-token=DO_ACCESS_TOKEN",
        "digitalocean-region=nyc2",
        "digitalocean-size=4gb",
        "digitalocean-private-networking",
        "engine-registry-mirror=http://10.11.12.13:12345"   # Docker Machine is using registry mirroring
    ]
  [runners.cache]
    Type = "s3"   # The Runner is using a distributed cache with Amazon S3 service
    ServerAddress = "s3-eu-west-1.amazonaws.com"
    AccessKey = "AMAZON_S3_ACCESS_KEY"
    SecretKey = "AMAZON_S3_SECRET_KEY"
    BucketName = "runners"
    Insecure = false
```

Note that the `MachineOptions` parameter contains options for the `digitalocean`
driver which is used by Docker Machine to spawn machines hosted on Digital Ocean,
and one option for Docker Machine itself (`engine-registry-mirror`).

## What are the supported cloud providers

The autoscale mechanism currently is based on Docker Machine. Advanced
configuration options, including virtualization/cloud provider parameters, are
available at the [Docker Machine documentation][docker-machine-driver].

[cache]: http://doc.gitlab.com/ce/ci/yaml/README.html#cache
[runner-installation]: https://gitlab.com/gitlab-org/gitlab-ci-multi-runner#installation
[runner-configuration]: https://gitlab.com/gitlab-org/gitlab-ci-multi-runner#advanced-configuration
[docker-machine-docs]: https://docs.docker.com/machine/
[docker-machine-driver]: https://docs.docker.com/machine/drivers/
[docker-machine-installation]: https://docs.docker.com/machine/install-machine/
[runners-cache]: advanced-configuration.md#the-runnerscache-section
[registry]: https://docs.docker.com/docker-trusted-registry/overview/
[caching]: ../install/autoscaling.md#install-the-cache-server
[registry-server]: ../install/autoscaling.md#install-docker-registry
