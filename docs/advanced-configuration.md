### Config file

Configuration uses TOML format described here: https://github.com/toml-lang/toml

1. The global section:
    ```
    concurrent = 4
    root_dir = ""
    ```
    
    This defines global settings of multi-runner:
    * `concurrent` - limits how many jobs globally can be run concurrently. The most upper limit of jobs using all defined runners
    * `root_dir` - allows to change relative dir where all builds, caches, etc. are stored. By default is current working directory

1. The [[runners]] section:
    ```
    [[runners]]
      name = "ruby-2.1-docker"
      url = "https://CI/"
      token = "TOKEN"
      limit = 0
      executor = "docker"
      builds_dir = ""
      shell = ""
      clean_environment = false
      environment = ["ENV=value", "LC_ALL=en_US.UTF-8"]
      disable_verbose = false
    ```

    This defines one runner entry:
    * `name` - not used, just informatory
    * `url` - CI URL
    * `token` - runner token
    * `limit` - limit how many jobs can be handled concurrently by this token. 0 simply means don't limit.
    * `executor` - select how project should be built. See below.
    * `builds_dir` - directory where builds will be stored in context of selected executor (Locally, Docker, SSH)
    * `clean_environment` - do not inherit any environment variables from the multi-runner process
    * `environment` - append or overwrite environment variables
    * `shell` - the name of shell to generate the script (default value is platform dependent)

1. The EXECUTORS:

    There are a couple of available executors currently:
    * **shell** - run build locally, default
    * **docker** - run build using Docker container - this requires the presence of *[runners.docker]*
    * **docker-ssh** - run build using Docker container, but connect to it with SSH - this requires the presence of *[runners.docker]* and *[runners.ssh]*
    * **ssh** - run build remotely with SSH - this requires the presence of *[runners.ssh]*
    * **parallels** - run build using Parallels VM, but connect to it with SSH - this requires the presence of *[runners.parallels]* and *[runners.ssh]*

1. The SHELLS:

    There are a couple of available shells that can be run on different platforms:
    * **bash** - generate Bash (Bourne-shell) script. All commands executed in Bash context (default for all Unix systems)
    * **cmd** - generate Windows Batch script. All commands are executed in Batch context (default for Windows)
    * **powershell** - generate Windows PowerShell script. All commands are executed in PowerShell context

1. The [runners.docker] section:
    ```
    [runners.docker]
      host = ""
      hostname = ""
      tls_cert_path = "/Users/ayufan/.boot2docker/certs"
      image = "ruby:2.1"
      privileged = false
      disable_cache = false
      disable_pull = false
      wait_for_services_timeout = 30
      cache_dir = ""
      registry = ""
      volumes = ["/data", "/home/project/cache"]
      extra_hosts = ["other-host:127.0.0.1"]
      links = ["mysql_container:mysql"]
      services = ["mysql", "redis:2.8", "postgres:9"]
    ```
    
    This defines the Docker Container parameters:
    * `host` - specify custom Docker endpoint, by default *DOCKER_HOST* environment is used or *"unix:///var/run/docker.sock"*
    * `hostname` - specify custom hostname for Docker container
    * `tls_cert_path` - when set it will use ca.pem, cert.pem and key.pem from that folder to make secure TLS connection to Docker (useful in boot2docker)
    * `image` - use this image to run builds
    * `privileged` - make container run in Privileged mode (insecure)
    * `disable_cache` - disable automatic
    * `disable_pull` - disable automatic image pulling if not found
    * `wait_for_services_timeout` - specify how long to wait for docker services, set to 0 to disable, default: 30
    * `cache_dir` - specify where Docker caches should be stored (this can be absolute or relative to current working directory)
    * `registry` - specify custom Docker registry to be used
    * `volumes` - specify additional volumes that should be cached
    * `extra_hosts` - specify hosts that should be defined in container environment
    * `links` - specify containers which should be linked with building container
    * `services` - specify additional services that should be run with build. Please visit [Docker Registry](https://registry.hub.docker.com/) for list of available applications. Each service will be run in separate container and linked to the build.

1. The [runners.parallels] section:
    ```
    [runners.parallels]
      base_name = "my-parallels-image"
      template_name = ""
      disable_snapshots = false
    ```

    This defines the Parallels parameters:
    * `base_name` - name of Parallels VM which will be cloned
    * `template_name` - custom name of Parallels VM linked template (optional)
    * `disable_snapshots` - if disabled the VMs will be destroyed after build

1. The [runners.ssh] section:
    ```
    [runners.ssh]
      host = "my-production-server"
      port = "22"
      user = "root"
      password = "production-server-password"
    ```
    
    This defines the SSH connection parameters:
    * `host` - where to connect (it's override when using *docker-ssh*)
    * `port` - specify port, default: 22
    * `user` - specify user
    * `password` - specify password
