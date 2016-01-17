FROM jpetazzo/dind

# Install Docker from Docker Inc. repositories.
RUN curl -sSL https://get.docker.com/ | sh && \
    apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y docker-engine && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

RUN curl -s -L https://github.com/docker/compose/releases/latest | \
    egrep -o '/docker/compose/releases/download/[0-9.]*/docker-compose-Linux-x86_64' | \
    wget --base=http://github.com/ -i - -O /usr/local/bin/docker-compose && \
    chmod +x /usr/local/bin/docker-compose && \
    /usr/local/bin/docker-compose --version

ENV LOG=file
ENTRYPOINT ["wrapdocker"]
CMD []
