#!/bin/bash

set -e
USER="$1"

status() {
	pidof gitlab-ci-multi-runner
}

echo Checking existence of $USER...
id -u "$USER"

echo Check if /etc/gitlab-runner/config.toml is created...
if [[ -f /etc/gitlab-runner/config.toml ]]; then
	CONFIG=$(ls -al /etc/gitlab-runner | grep config.toml)
	echo $CONFIG | grep "\-rw-------"
	echo $CONFIG | grep "root root"
fi

echo List of processes:
ps auxf
echo

echo Checking if runner is running...
status
echo

echo Testing help...
gitlab-runner --help > /dev/null
echo

echo Stopping runner...
gitlab-runner stop
! status
echo

echo Starting runner...
gitlab-runner start
sleep 1s
status
echo

echo Registering runner...
gitlab-runner register -n -u https://ci.gitlab.com/ -r 51cc1fe15c0ac7d3d5564e24ada4d1
echo

echo Checking su...
echo id | su --shell /bin/bash --login "$USER"
