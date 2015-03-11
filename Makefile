NAME := gitlab-ci-multi-runner
REVISION := $(shell git rev-parse --short HEAD || echo unknown)
VERSION := $(shell git describe --tags || cat VERSION || echo dev)
VERSION := $(shell echo $(VERSION) | sed -e 's/^v//g')
ITTERATION := $(shell date +%s)

all: build

build:
	gox -os="linux" -os="darwin" -os="windows" -output="out/{{.Dir}}-{{.OS}}-{{.Arch}}"

test:
	go test

deploy:
	gox -osarch="linux/amd64" -output="out/{{.Dir}}-{{.OS}}-{{.Arch}}"
	scp out/gitlab-ci-multi-runner-linux-amd64 lab-worker:
	ssh lab-worker ./gitlab-ci-multi-runner-linux-amd64 --debug run

deploy2:
	gox -osarch="linux/amd64" -output="out/{{.Dir}}-{{.OS}}-{{.Arch}}"
	scp out/gitlab-ci-multi-runner-linux-amd64 gitlab_ci_runner@lab-docker:
	ssh gitlab_ci_runner@lab-docker ./gitlab-ci-multi-runner-linux-amd64 --debug run

version: FORCE
	echo "package commands\n\nconst VERSION = \"$(VERSION) ($(REVISION))\"\nconst REVISION = \"$(REVISION)\"" > commands/version.go

deb:
	make build-deb ARCH=amd64 TYPE=sysv
	make build-deb ARCH=386 TYPE=sysv
	make build-deb ARCH=amd64 TYPE=upstart
	make build-deb ARCH=386 TYPE=upstart
	make build-deb ARCH=amd64 TYPE=systemd
	make build-deb ARCH=386 TYPE=systemd

rpm:
	make build-rpm ARCH=amd64 TYPE=systemd
	make build-rpm ARCH=386 TYPE=systemd

build-deb:
	mkdir -p out/deb/$(TYPE)/$(ARCH)/
	fpm -s dir -t deb -n $(NAME) -v $(VERSION) \
		-p out/deb/$(TYPE)/$(ARCH)/$(NAME).deb \
		--deb-priority optional --category admin \
		--force \
		--deb-compression bzip2 \
		--after-install packaging/$(TYPE)/scripts/postinst.deb \
		--before-remove packaging/$(TYPE)/scripts/prerm.deb \
		--url https://github.com/ayufan/gitlab-ci-multi-runner \
		--description "GitLab CI Multi-purpose Runner" \
		-m "Kamil Trzciński <ayufan@ayufan.eu>" \
		--license "MIT" \
		--vendor "ayufan.eu" \
		-a $(ARCH) \
		--config-files /etc/default/gitlab-ci-multi-runner \
		out/gitlab-ci-multi-runner-linux-$(ARCH)=/usr/bin/gitlab-ci-multi-runner \
		packaging/$(TYPE)/root/=/

build-rpm:
	mkdir -p out/rpm/$(TYPE)/$(ARCH)/
	fpm -s dir -t rpm -n $(NAME) -v $(VERSION) \
		-p out/rpm/$(TYPE)/$(ARCH)/$(NAME).rpm \
		--rpm-compression bzip2 --rpm-os linux \
		--force \
		--after-install packaging/$(TYPE)/scripts/postinst.rpm \
		--before-remove packaging/$(TYPE)/scripts/prerm.rpm \
		--url https://github.com/ayufan/gitlab-ci-multi-runner \
		--description "GitLab CI Multi-purpose Runner" \
		-m "Kamil Trzciński <ayufan@ayufan.eu>" \
		--license "MIT" \
		--vendor "ayufan.eu" \
		-a $(ARCH) \
		out/gitlab-ci-multi-runner-linux-$(ARCH)=/usr/bin/gitlab-ci-multi-runner \
		packaging/$(TYPE)/root/=/

packagecloud-deb:
	package_cloud push ayufan/gitlab-ci-multi-runner/debian/wheezy out/deb/sysv/*/*.deb
	package_cloud push ayufan/gitlab-ci-multi-runner/debian/jessie out/deb/systemd/*/*.deb

	package_cloud push ayufan/gitlab-ci-multi-runner/ubuntu/precise out/deb/upstart/*/*.deb
	package_cloud push ayufan/gitlab-ci-multi-runner/ubuntu/trusty out/deb/upstart/*/*.deb
	package_cloud push ayufan/gitlab-ci-multi-runner/ubuntu/utopic out/deb/sysv/*/*.deb

packagecloud-rpm:
	package_cloud push ayufan/gitlab-ci-multi-runner/el/7 out/rpm/systemd/*/*.rpm

FORCE:
