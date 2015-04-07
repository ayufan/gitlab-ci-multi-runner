#!/bin/bash

set -e

finish() {
	trap - EXIT
	echo -----------------
	echo "$@"
	cat /var/log/gitlab-ci-multi-runner.log || true
	cat /var/log/upstart/gitlab-ci-multi-runner.log || true
	kill -s SIGABRT 1
	kill -s SIGILL 1
}

if [[ -e /bin/pidof ]]; then
	status() {
		/bin/pidof gitlab-ci-multi-runner
	}
else
	status() {
		ps aux | grep "gitlab-ci-multi-runner run"
	}
fi

before_script() {
	trap 'set +x; finish RunnerTestFailed' EXIT
	set -x
	sleep 5s
}

test_script() {
	sleep 1s
	[[ -e /bin/ps ]] && ps auxf

	gitlab-ci-multi-runner --help
	status

	gitlab-ci-multi-runner stop
	sleep 1s
	! status

	gitlab-ci-multi-runner start
	sleep 1s
	status

	gitlab-ci-multi-runner uninstall
	sleep 1s
	! status

	set +x
	finish RunnerTestSuccess
}

test_redhat() {
	before_script
	yum install -y /out/rpm/*amd64.rpm
	test_script
}

test_debian() {
	before_script
	dpkg -i /out/deb/*amd64.deb
	test_script
}

if [[ -e /etc/redhat-release ]]; then
	test_redhat &
elif [[ -e /etc/debian_version ]]; then
	test_debian &
else
	cat >&2 <<'EOF'
Test platform is not detectable. Please hack tests/install_runner.sh.
EOF
	exit 1
fi

exec /sbin/init
