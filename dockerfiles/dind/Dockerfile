FROM jpetazzo/dind

RUN apt-get update && \
    apt-get upgrade -y
RUN apt-get install -y lxc-docker

RUN curl -s -L https://github.com/docker/compose/releases/latest | \
    egrep -o '/docker/compose/releases/download/[0-9.]*/docker-compose-Linux-x86_64' | \
    wget --base=http://github.com/ -i - -O /usr/local/bin/docker-compose && \
    chmod +x /usr/local/bin/docker-compose && \
    /usr/local/bin/docker-compose --version

ENV LOG=file
ENTRYPOINT ["wrapdocker"]
CMD []
