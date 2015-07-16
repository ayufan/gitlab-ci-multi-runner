FROM ubuntu:14.04

RUN apt-get update -y && \
    apt-get upgrade -y && \
    apt-get install ca-certificates wget apt-transport-https vim nano

RUN echo "deb https://packages.gitlab.com/runner/gitlab-ci-multi-runner/ubuntu/ `lsb_release -cs` main" > /etc/apt/sources.list.d/runner_gitlab-ci-multi-runner.list && \
    wget -q -O - https://packages.gitlab.com/gpg.key | apt-key add - && \
    apt-get update -y && \
    apt-get install -y gitlab-ci-multi-runner

VOLUME ["/etc/gitlab-runner", "/home/gitlab-runner"]
ENTRYPOINT ["gitlab-ci-multi-runner"]
CMD ["run", "--user=gitlab-runner", "--working-directory=/home/gitlab-runner"]
