### Bleeding edge releases (development)

1. Download one of the binaries:

* https://repo.ayufan.eu/gitlab-ci-multi-runner/master/binaries/gitlab-ci-multi-runner-linux-386
* https://repo.ayufan.eu/gitlab-ci-multi-runner/master/binaries/gitlab-ci-multi-runner-linux-amd64
* https://repo.ayufan.eu/gitlab-ci-multi-runner/master/binaries/gitlab-ci-multi-runner-linux-arm
* https://repo.ayufan.eu/gitlab-ci-multi-runner/master/binaries/gitlab-ci-multi-runner-darwin-386
* https://repo.ayufan.eu/gitlab-ci-multi-runner/master/binaries/gitlab-ci-multi-runner-darwin-amd64
* https://repo.ayufan.eu/gitlab-ci-multi-runner/master/binaries/gitlab-ci-multi-runner-windows-386.exe
* https://repo.ayufan.eu/gitlab-ci-multi-runner/master/binaries/gitlab-ci-multi-runner-windows-amd64.exe

You can then run the runner with:
```bash
chmod +x gitlab-ci-multi-runner-linux-amd64
./gitlab-ci-multi-runner-linux-amd64 run
```

1. Download one of the packages for Debian or Ubuntu:

* https://repo.ayufan.eu/gitlab-ci-multi-runner/master/deb/gitlab-ci-multi-runner_386.deb
* https://repo.ayufan.eu/gitlab-ci-multi-runner/master/deb/gitlab-ci-multi-runner_amd64.deb
* https://repo.ayufan.eu/gitlab-ci-multi-runner/master/deb/gitlab-ci-multi-runner_arm.deb

You can then install it with:
```bash
dpkg -i gitlab-ci-multi-runner_386.deb
```

1. Download one of the packages for RedHat or CentOS:

* https://repo.ayufan.eu/gitlab-ci-multi-runner/master/rpm/gitlab-ci-multi-runner_386.rpm
* https://repo.ayufan.eu/gitlab-ci-multi-runner/master/rpm/gitlab-ci-multi-runner_amd64.rpm

You can then install it with:
```bash
rpm -i gitlab-ci-multi-runner_386.rpm
```

1. Download any other tagged release:

Simple replace the `master` with either `tag` (v0.2.0) or `latest` (the latest stable).

* https://repo.ayufan.eu/gitlab-ci-multi-runner/latest/binaries/gitlab-ci-multi-runner-linux-386
* https://repo.ayufan.eu/gitlab-ci-multi-runner/v0.2.0/binaries/gitlab-ci-multi-runner-linux-386

If you have problem downloading fallback to http://:

* http://repo.ayufan.eu/gitlab-ci-multi-runner/latest/binaries/gitlab-ci-multi-runner-linux-386
* http://repo.ayufan.eu/gitlab-ci-multi-runner/v0.2.0/binaries/gitlab-ci-multi-runner-linux-386
