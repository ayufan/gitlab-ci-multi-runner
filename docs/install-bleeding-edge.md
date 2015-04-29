### Bleeding edge releases (development)

1. Download one of the binaries:

* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/binaries/gitlab-ci-multi-runner-linux-386
* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/binaries/gitlab-ci-multi-runner-linux-amd64
* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/binaries/gitlab-ci-multi-runner-linux-arm
* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/binaries/gitlab-ci-multi-runner-darwin-386
* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/binaries/gitlab-ci-multi-runner-darwin-amd64
* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/binaries/gitlab-ci-multi-runner-windows-386.exe
* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/binaries/gitlab-ci-multi-runner-windows-amd64.exe

You can then run the runner with:
```bash
chmod +x gitlab-ci-multi-runner-linux-amd64
./gitlab-ci-multi-runner-linux-amd64 run
```

1. Download one of the packages for Debian or Ubuntu:

* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/deb/gitlab-ci-multi-runner_386.deb
* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/deb/gitlab-ci-multi-runner_amd64.deb
* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/deb/gitlab-ci-multi-runner_arm.deb

You can then install it with:
```bash
dpkg -i gitlab-ci-multi-runner_386.deb
```

1. Download one of the packages for RedHat or CentOS:

* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/rpm/gitlab-ci-multi-runner_386.rpm
* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/rpm/gitlab-ci-multi-runner_amd64.rpm

You can then install it with:
```bash
rpm -i gitlab-ci-multi-runner_386.rpm
```

1. Download any other tagged release:

Simple replace the `master` with either `tag` (v0.2.0) or `latest` (the latest stable).

* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-ci-multi-runner-linux-386
* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/v0.2.0/binaries/gitlab-ci-multi-runner-linux-386

If you have problem downloading fallback to http://:

* http://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-ci-multi-runner-linux-386
* http://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/v0.2.0/binaries/gitlab-ci-multi-runner-linux-386
