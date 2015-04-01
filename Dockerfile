# We use the onbuild version of library/golang,
# (https://github.com/docker-library/golang/blob/master/1.4/onbuild/Dockerfile)
FROM golang:onbuild

# Set working directory to /data
# 
# this is where the config.toml will be sourced,
# you can mount your own custom data directory at /data
WORKDIR /data
VOLUME /data

# init sets up the environment and launches gitlab-ci-multi-runner
ENTRYPOINT ["/go/src/app/packaging/docker/init"]
CMD ["run"]
