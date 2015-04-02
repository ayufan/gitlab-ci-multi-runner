NAME := gitlab-ci-multi-runner
REVISION := $(shell git rev-parse --short HEAD || echo unknown)
VERSION := $(shell git describe --tags || cat VERSION || echo dev)
VERSION := $(shell echo $(VERSION) | sed -e 's/^v//g')
ITTERATION := $(shell date +%s)
PACKAGE_CLOUD ?= ayufan/gitlab-ci-multi-runner
BUILD_PLATFORMS ?= -os="linux" -os="darwin" -os="windows"

all: deps test lint toolchain build

help:
	# make all => deps test lint toolchain build
	# make deps - install all dependencies
	# make test - run project tests
	# make lint - check project code style
	# make toolchain - install crossplatform toolchain
	# make build - build project for all supported OSes
	# make package - package project using FPM
	# make packagecloud - send all packages to packagecloud
	# make packagecloud-yank - remove specific version from packagecloud

deps:
	# Installing dependencies...
	go get github.com/tools/godep
	go get -u github.com/golang/lint/golint
	go get github.com/mitchellh/gox
	-go get code.google.com/p/winsvc/eventlog
	godep restore

toolchain:
	# Building toolchain...
	gox -build-toolchain $(BUILD_PLATFORMS)

build: version
	# Building gitlab-ci-multi-runner for $(BUILD_PLATFORMS)
	gox $(BUILD_PLATFORMS) -output="out/{{.Dir}}-{{.OS}}-{{.Arch}}"

lint:
	# Checking project code style...
	golint ./... | grep -v "be unexported"

test:
	# Running tests...
	go test

version: FORCE
	# Generating VERSION...
	echo "package commands\n\nconst VERSION = \"$(VERSION) ($(REVISION))\"\nconst REVISION = \"$(REVISION)\"" > commands/version.go

package: package-deps package-deb package-rpm

package-deb:
	# Building Debian compatible packages...
	make package-deb-fpm ARCH=amd64 TYPE=sysv
	make package-deb-fpm ARCH=386 TYPE=sysv
	make package-deb-fpm ARCH=amd64 TYPE=upstart
	make package-deb-fpm ARCH=386 TYPE=upstart
	make package-deb-fpm ARCH=amd64 TYPE=systemd
	make package-deb ARCH=386 TYPE=systemd

package-rpm:
	# Building RedHat compatible packages...
	make package-rpm-fpm ARCH=amd64 TYPE=systemd
	make package-rpm-fpm ARCH=386 TYPE=systemd

package-deps:
	# Installing packaging dependencies...
	gem install fpm

package-deb-fpm:
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

package-rpm-fpm:
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

packagecloud: packagecloud-deps packagecloud-deb packagecloud-rpm

packagecloud-deps:
	# Installing packagecloud dependencies...
	gem install package_cloud

packagecloud-deb:
	# Sending Debian compatible packages...
	package_cloud push $(PACKAGE_CLOUD)/debian/wheezy out/deb/sysv/*/*.deb
	package_cloud push $(PACKAGE_CLOUD)/debian/jessie out/deb/systemd/*/*.deb

	package_cloud push $(PACKAGE_CLOUD)/ubuntu/precise out/deb/upstart/*/*.deb
	package_cloud push $(PACKAGE_CLOUD)/ubuntu/trusty out/deb/upstart/*/*.deb
	package_cloud push $(PACKAGE_CLOUD)/ubuntu/utopic out/deb/sysv/*/*.deb

packagecloud-rpm:
	# Sending RedHat compatible packages...
	package_cloud push $(PACKAGE_CLOUD)/el/7 out/rpm/systemd/*/*.rpm

packagecloud-yank:
ifneq ($(YANK),)
	# Removing $(YANK) from packagecloud...
	-for DIST in debian/wheezy debian/jessie ubuntu/precise ubuntu/trusty ubuntu/utopic; do \
		package_cloud yank $(PACKAGE_CLOUD)/$$DIST $(NAME)_$(YANK)_amd64.deb; \
		package_cloud yank $(PACKAGE_CLOUD)/$$DIST $(NAME)_$(YANK)_386.deb; \
	done
	-package_cloud yank $(PACKAGE_CLOUD)/el/7 $(NAME)-$(YANK)-1.x86_64.rpm
	-package_cloud yank $(PACKAGE_CLOUD)/el/7 $(NAME)-$(YANK)-1.386.rpm
else
	# No version specified in YANK
	@exit 1
endif

FORCE:
