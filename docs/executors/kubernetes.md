# The Kubernetes executor (**EXPERIMENTAL**)

GitLab Runner can use Kubernetes to run builds on a kubernetes cluster. This is
possible with the use of the **Kubernetes** executor.

The **Kubernetes** executor, when used with GitLab CI, connects to the Kubernetes
API in the cluster creating a Pod for each GitLab CI Job. This Pod is made
up of, at the very least, a build container, there will
then be additional containers, one for each `service` defined by the GitLab CI
yaml. The names for these containers are as follows:

- The build container is `build`
- The services containers are `svc-X` where `X` is `[0-9]+`

---

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Workflow

The Kubernetes executor divides the build into multiple steps:

1. **Prepare**: Create the Pod against the Kubernetes Cluster.
	This creates the containers required for the build and services to run.
1. **Pre-build**: Clone, restore cache and download artifacts from previous
   stages.
   User provided image needs to have `git` installed.
1. **Build**: User build.
1. **Post-build**: Create cache, upload artifacts to GitLab.

All stages are run on user provided image. 
This image needs to have `git` installed and optionally
GitLab Runner binary installed for supporting artifacts and caching.

## Connecting to the Kubernetes API

The following options are provided, which allow you to connect to the Kubernetes API:

- `host`: Optional Kubernetes master host URL (auto-discovery attempted if not specified)
- `cert_file`: Optional Kubernetes master auth certificate
- `key_file`: Optional Kubernetes master auth private key
- `ca_file`: Optional Kubernetes master auth ca certificate

If you are running the GitLab CI Runner within the Kubernetes cluster you can omit
all of the above fields to have the Runner auto-discovery the Kubernetes API. This
is the recommended approach.

If you are running it externally to the Cluster then you will need to set each
of these keywords and make sure that the Runner has access to the Kubernetes API
on the cluster.

## The keywords

The following keywords help to define the behaviour of the Runner within kubernetes:

- `namespace`: Namespace to run Kubernetes Pods in
- `privileged`: Run containers with the privileged flag
- `cpus`: The CPU allocation given to build containers
- `memory`: The amount of memory allocated to build containers
- `service_cpus`: The CPU allocation given to build service containers
- `service_memory`: The amount of memory allocated to build service containers

## Define keywords in the config toml

Each of the keywords can be defined in the `config.toml` for the gitlab runner.

Here is an example `config.toml`:

```toml
concurrent = 4

[[runners]]
  name = "Kubernetes Runner"
  url = "https://gitlab.com/ci"
  token = "......"
  executor = "kubernetes"
  [runners.kubernetes]
    host = "https://45.67.34.123:4892"
    cert_file = "/etc/ssl/kubernetes/api.crt"
    key_file = "/etc/ssl/kubernetes/api.key"
    ca_file = "/etc/ssl/kubernetes/ca.crt"
    namespace = "gitlab"
    privileged = true
    cpus = "750m"
    memory = "250m"
    service_cpus = "1000m"
    service_memory = "450m"
```
