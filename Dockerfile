# This image is used to create bleeding edge docker image and is not compatible with any other image
FROM golang:1.6

# Copy sources
COPY . /go/src/gitlab.com/gitlab-org/gitlab-ci-multi-runner
WORKDIR /go/src/gitlab.com/gitlab-org/gitlab-ci-multi-runner

# Fetch tags (to have proper versioning)
RUN git fetch --tags || true

# Build development version
ENV BUILD_PLATFORMS -osarch=linux/amd64
RUN make && \
	ln -s $(pwd)/out/binaries/gitlab-ci-multi-runner-linux-amd64 /usr/bin/gitlab-ci-multi-runner && \
	ln -s $(pwd)/out/binaries/gitlab-ci-multi-runner-linux-amd64 /usr/bin/gitlab-runner

# Install runner
RUN packaging/root/usr/share/gitlab-runner/post-install

# Preserve runner's data
VOLUME ["/etc/gitlab-runner", "/home/gitlab-runner"]

# init sets up the environment and launches gitlab-runner
CMD ["run", "--user=gitlab-runner", "--working-directory=/home/gitlab-runner"]
ENTRYPOINT ["/usr/bin/gitlab-runner"]
