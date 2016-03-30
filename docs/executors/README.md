# Executors

GitLab Runner implements a number of executors that can be used to run your
builds in different scenarios:

- [Shell](shell.md)
- [Docker](docker.md)
- [Docker Machine and Docker Machine SSH (auto-scaling)](../install/autoscaling.md)
- [Parallels](parallels.md)
- [VirtualBox](virtualbox.md)
- [SSH](ssh.md)

## Select the executor

The executors support different platforms and methodologies for building a
project. The table below shows the key facts for each executor which will help
you decide.

| Executor                                          | Shell   | Docker | Docker-SSH | VirtualBox | Parallels | SSH  |
|---------------------------------------------------|---------|--------|------------|------------|-----------|------|
| Clean build environment for every build           | no      | ✓      | ✓          | ✓          | ✓         | no   |
| Migrate runner machine                            | no      | ✓      | ✓          | partial    | partial   | no   |
| Zero-configuration support for concurrent builds  | no (1)  | ✓      | ✓          | ✓          | ✓         | no   |
| Complicated build environments                    | no (2)  | ✓      | ✓          | ✓ (3)      | ✓ (3)     | no   |
| Debugging build problems                          | easy    | medium | medium     | hard       | hard      | easy |

1. it's possible, but in most cases it is problematic if the build uses services
   installed on the build machine
2. it requires to install all dependencies by hand
3. for example using Vagrant

### I'm not sure

**Shell** is the simplest executor to configure. All required dependencies for
your builds need to be installed manually on the machine that the Runner is
installed.

A better way is to use **Docker** as it allows to have a clean build environment,
with easy dependency management (all dependencies for building the project could
be put in the Docker image). The Docker executor allows you to easily create
a build environment with dependent [services], like MySQL.

We usually don't advise to use **Docker-SSH** which is a special version of
the **Docker** executor. This executor allows you to connect to a Docker
container that runs the **SSH** daemon inside it. It can be useful if your
Docker image tries to replicate a full working system: it uses some process
management system (`init`), exposes the SSH daemon, and contains already
installed services. These kind of images are fat images, and are not generally
advised to be used by the Docker community.

We also offer two full system virtualization options: **VirtualBox** and
**Parallels**. This type of executor allows you to use an already created
virtual machine, which will be cloned and used to run your build. It can prove
useful if you want to run your builds on different Operating Systems since it
allows to create virtual machines with Windows, Linux, OSX or FreeBSD and make
GitLab Runner to connect to the virtual machine and run the build on it. Its
usage can also be useful to reduce the cost of infrastructure.

The **SSH** executor is added for completeness. It's the least supported
executor from all of the already mentioned ones. It makes GitLab Runner to
connect to some external server and run the builds there. We have some success
stories from organizations using that executor, but generally we advise to use
any of the above.

[services]: http://doc.gitlab.com/ce/ci/services/README.html
