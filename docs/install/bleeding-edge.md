## Bleeding edge releases (development)

### Download the standalone binaries

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

### Download one of the packages for Debian or Ubuntu

* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/deb/gitlab-ci-multi-runner_i386.deb
* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/deb/gitlab-ci-multi-runner_amd64.deb
* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/deb/gitlab-ci-multi-runner_arm.deb
* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/deb/gitlab-ci-multi-runner_armhf.deb

You can then install it with:
```bash
dpkg -i gitlab-ci-multi-runner_386.deb
```

### Download one of the packages for RedHat or CentOS

* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/rpm/gitlab-ci-multi-runner_i686.rpm
* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/rpm/gitlab-ci-multi-runner_amd64.rpm
* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/rpm/gitlab-ci-multi-runner_arm.rpm
* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/rpm/gitlab-ci-multi-runner_armhf.rpm

You can then install it with:
```bash
rpm -i gitlab-ci-multi-runner_386.rpm
```

### Download any other tagged release

Simply replace `master` with either `tag` (v0.2.0 or 0.4.2) or `latest` (the latest
stable). For a list of tags see <https://gitlab.com/gitlab-org/gitlab-ci-multi-runner/tags>.
For example:

* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/binaries/gitlab-ci-multi-runner-linux-386
* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-ci-multi-runner-linux-386
* https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/v0.2.0/binaries/gitlab-ci-multi-runner-linux-386

If you have problem downloading through `https`, fallback to plain `http`:

* http://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/binaries/gitlab-ci-multi-runner-linux-386
* http://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-ci-multi-runner-linux-386
* http://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/v0.2.0/binaries/gitlab-ci-multi-runner-linux-386
