#!/bin/bash

if [[ $# -lt 2 ]]; then
	echo "usage $0: <image> <file> <upgrade>"
	exit 1
fi

set -xe
set -o pipefail

IMAGE="$1"
INSTALL_FILE="$2"
UPGRADE="$3"

if [[ ! -f "$INSTALL_FILE" ]]; then
	echo "$INSTALL_FILE: not found"
	exit 1
fi

ID=gitlab_runner
docker rm -f -v $ID || :
docker run -d --name $ID --privileged "$IMAGE" /sbin/init
# trap "docker rm -f -v $ID" EXIT
cat "$INSTALL_FILE" | docker exec -i $ID bash -c "cat > /$(basename $INSTALL_FILE)"

case $IMAGE in
	debian:*|ubuntu-upstart:*)
		docker exec $ID apt-get update -y
		docker exec $ID apt-get install -y curl procps
		if [[ -n "$UPGRADE" ]]; then
			curl -L https://packages.gitlab.com/install/repositories/runner/gitlab-ci-multi-runner/script.deb.sh | docker exec -i $ID bash
			docker exec $ID apt-get install -y gitlab-ci-multi-runner
		fi
		if ! docker exec $ID dpkg -i "/$(basename $INSTALL_FILE)"
		then
			docker exec $ID apt-get install -f -y
			docker exec $ID dpkg -i "/$(basename $INSTALL_FILE)"
		fi
		;;

	centos:*)
		docker exec $ID yum install -y curl sysvinit-tools
		if [[ -n "$UPGRADE" ]]; then
			curl -L https://packages.gitlab.com/install/repositories/runner/gitlab-ci-multi-runner/script.rpm.sh | docker exec -i $ID bash
			docker exec $ID yum install -y gitlab-ci-multi-runner
		fi
		docker exec $ID yum localinstall -y "/$(basename $INSTALL_FILE)"
		;;

	*)
		echo "ERROR: unsupported $IMAGE."
esac

if [[ -n "$UPGRADE" ]]; then
	USER="gitlab_ci_multi_runner"
else
	USER="gitlab-runner"
fi

cat $(dirname $0)/test_script.sh | docker exec -i $ID bash -s "$USER"
