# This image is used to create bleeding edge docker image and is not compatible with any other image
FROM golang:onbuild

# Install runner
RUN /go/src/app/dockerfiles/packaging/root/usr/share/gitlab-runner/post-install

# Preserve runner's data
VOLUME ["/etc/gitlab-runner", "/home/gitlab-runner"]

# init sets up the environment and launches gitlab-runner
ENTRYPOINT ["/go/src/app/ubuntu/entrypoint"]
CMD ["run", "--user=gitlab-runner", "--working-directory=/home/gitlab-runner"]
