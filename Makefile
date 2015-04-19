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
	gox $(BUILD_PLATFORMS) -output="out/binaries/{{.Dir}}-{{.OS}}-{{.Arch}}"

lint:
	# Checking project code style...
	golint ./... | grep -v "be unexported"

test:
	# Running tests...
	go test

test-docker:
	make test-docker-image IMAGE=centos:6 CMD="yum install -y tar &&"
	#make test-docker-image IMAGE=centos:7 CMD="yum install -y tar &&"
	make test-docker-image IMAGE=debian:wheezy
	make test-docker-image IMAGE=debian:jessie
	make test-docker-image IMAGE=ubuntu-upstart:precise
	make test-docker-image IMAGE=ubuntu-upstart:trusty
	make test-docker-image IMAGE=ubuntu-upstart:utopic

test-docker-image:
	-tar c tests/* out/*/* | docker run -P --rm -i $(IMAGE) bash -c "$(CMD) tar x && exec tests/install_runner.sh"

version: FORCE
	# Generating VERSION...
	echo "package commands\n\nconst VERSION = \"$(VERSION) ($(REVISION))\"\nconst REVISION = \"$(REVISION)\"" > commands/version.go

package: package-deps package-deb package-rpm package-script

package-script:
	cp install.sh out/
	[[ -n "$TRAVIS_TAG" ]] || sed "s|/latest/|/master/|g" install.sh > out/install.sh

package-deb:
	# Building Debian compatible packages...
	make package-deb-fpm ARCH=amd64
	make package-deb-fpm ARCH=386

package-rpm:
	# Building RedHat compatible packages...
	make package-rpm-fpm ARCH=amd64

package-deps:
	# Installing packaging dependencies...
	gem install fpm

package-deb-fpm:
	@mkdir -p out/deb/
	fpm -s dir -t deb -n $(NAME) -v $(VERSION) \
		-p out/deb/$(NAME)_$(ARCH).deb \
		--deb-priority optional --category admin \
		--force \
		--deb-compression bzip2 \
		--after-install packaging/scripts/postinst.deb \
		--before-remove packaging/scripts/prerm.deb \
		--url https://github.com/ayufan/gitlab-ci-multi-runner \
		--description "GitLab CI Multi-purpose Runner" \
		-m "Kamil Trzciński <ayufan@ayufan.eu>" \
		--license "MIT" \
		--vendor "ayufan.eu" \
		-a $(ARCH) \
		out/binaries/gitlab-ci-multi-runner-linux-$(ARCH)=/usr/bin/gitlab-ci-multi-runner

package-rpm-fpm:
	@mkdir -p out/rpm/
	fpm -s dir -t rpm -n $(NAME) -v $(VERSION) \
		-p out/rpm/$(NAME)_$(ARCH).rpm \
		--rpm-compression bzip2 --rpm-os linux \
		--force \
		--after-install packaging/scripts/postinst.rpm \
		--before-remove packaging/scripts/prerm.rpm \
		--url https://github.com/ayufan/gitlab-ci-multi-runner \
		--description "GitLab CI Multi-purpose Runner" \
		-m "Kamil Trzciński <ayufan@ayufan.eu>" \
		--license "MIT" \
		--vendor "ayufan.eu" \
		-a $(ARCH) \
		out/binaries/gitlab-ci-multi-runner-linux-$(ARCH)=/usr/bin/gitlab-ci-multi-runner

packagecloud: packagecloud-deps packagecloud-deb packagecloud-rpm

packagecloud-deps:
	# Installing packagecloud dependencies...
	gem install package_cloud

packagecloud-deb:
	# Sending Debian compatible packages...
	package_cloud push $(PACKAGE_CLOUD)/debian/wheezy out/deb/*.deb
	package_cloud push $(PACKAGE_CLOUD)/debian/jessie out/deb/*.deb

	package_cloud push $(PACKAGE_CLOUD)/ubuntu/precise out/deb/*.deb
	package_cloud push $(PACKAGE_CLOUD)/ubuntu/trusty out/deb/*.deb
	package_cloud push $(PACKAGE_CLOUD)/ubuntu/utopic out/deb/*.deb

packagecloud-rpm:
	# Sending RedHat compatible packages...
	package_cloud push $(PACKAGE_CLOUD)/el/7 out/rpm/*.rpm

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
