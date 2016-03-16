# Executors

GitLab Runner implements a number of executors that can be used to run your builds in different scenarios:

* [Shell](shell.md)
* [Docker and Docker-SSH](docker.md)
* [Parallels](parallels.md)
* [VirtualBox](virtualbox.md)
* [SSH](ssh.md)

## Select the executor

The executors are created to support different and methodologies for building the project.

The table tries to answer about each of the key facts about using different executors:

| Executor                                               | Shell   | Docker | Docker-SSH | VirtualBox | Parallels | SSH  |
|--------------------------------------------------------|---------|--------|------------|------------|-----------|------|
| Clean build environment for every build                | no      | ✓      | ✓          | ✓          | ✓         | no   |
| Migrate runner machine                                 | no      | yes    | yes        | partial    | partial   | no   |
| Zero-configuration support for concurrent builds       | no (1)  | ✓      | ✓          | ✓          | ✓         | no   |
| Complicated build environments                         | no (2)  | ✓      | ✓          | ✓ (3)      | ✓ (3)     | no   |
| Debugging build problems                               | easy    | medium | medium     | hard       | hard      | easy |

1: it's possible, but in most cases it is problematic if build uses services installed on machine
2: it requires to install all dependencies by hand
3: for example using Vagrant

### I'm not sure

In most cases the best is to use **Shell** as this is the simplest to configure.
You simply have to install, and register GitLab Runner.
All required dependencies needs to be installed by hand using, ex.: `apt-get`.

The better way is to use **Docker** it allows to have a clean build environment,
with easy dependencies management (all dependencies for building the project could be put in Docker Image).
However, sometimes it can be complicated to debug potential problems with build.
The **Docker** allows to fairly easy create a build environments with dependent services, like: MySQL.

We usually don't advise to use **Docker-SSH** which is the special version of **Docker** executor.
It allows to connect to Docker Container that runs **SSHD** daemon inside.
This executor can be useful if you Docker Image tries to replicate full working system:
it uses some process management system (init 1), exposes SSHD daemon, and contains already installed services.
This kind of images are generally are fat images, and not generally advised to be used by Docker community.

We also offer two full system virtualisation options: **VirtualBox** and **Parallels**.
It allows you to use already created VM, which will be cloned and run for time of your build.
It can be useful if you require to use completely different system on your GitLab Runner machine.
Basically, it allows to create VM with Windows, OSX or FreeBSD and make GitLab Runner to connect to the VM
and run build on it. It can be useful to reduce cost of infrastructure.

The **SSH** is added for completeness. It's the least supported executor from all of the already mentioned.
It makes the GitLab Runner to connect to some external server and run builds there.
We have some success stories from organizations using that executor, but generally we advise to use any of the above.
