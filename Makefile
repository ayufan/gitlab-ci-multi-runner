NAME ?= gitlab-ci-multi-runner
ifeq ($(RELEASE),true)
	PACKAGE_NAME ?= $(NAME)
	PACKAGE_CONFLICT ?= $(NAME)-beta
else
	PACKAGE_NAME ?= $(NAME)-beta
	PACKAGE_CONFLICT ?= $(NAME)
endif
REVISION := $(shell git rev-parse --short HEAD || echo unknown)
VERSION := $(shell git describe --tags || cat VERSION || echo dev)
VERSION := $(shell echo $(VERSION) | sed -e 's/^v//g')
ITTERATION := $(shell date +%s)
PACKAGE_CLOUD ?= ayufan/gitlab-ci-multi-runner
PACKAGE_CLOUD_URL ?= https://packagecloud.io/
BUILD_PLATFORMS ?= -os="linux" -os="darwin" -os="windows"
S3_UPLOAD_PATH ?= master
DEB_PLATFORMS ?= debian/wheezy debian/jessie ubuntu/precise ubuntu/trusty ubuntu/utopic ubuntu/vivid
DEB_ARCHS ?= amd64 386 arm armhf
RPM_PLATFORMS ?= el/6 el/7 ol/6 ol/7
RPM_ARCHS ?= amd64 386 arm armhf

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
	gox $(BUILD_PLATFORMS) -output="out/binaries/$(NAME)-{{.OS}}-{{.Arch}}"

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
	echo "package common\n\nconst NAME = \"$(PACKAGE_NAME)\"\nconst VERSION = \"$(VERSION)\"\nconst REVISION = \"$(REVISION)\"" > common/version.go

package: package-deps package-deb package-rpm

package-deb:
	# Building Debian compatible packages...
	make package-deb-fpm ARCH=amd64 PACKAGE_ARCH=amd64
	make package-deb-fpm ARCH=i386 PACKAGE_ARCH=i386
	make package-deb-fpm ARCH=arm PACKAGE_ARCH=arm
	make package-deb-fpm ARCH=arm PACKAGE_ARCH=armhf

package-rpm:
	# Building RedHat compatible packages...
	make package-rpm-fpm ARCH=amd64 PACKAGE_ARCH=amd64
	make package-rpm-fpm ARCH=386 PACKAGE_ARCH=386
	make package-rpm-fpm ARCH=arm PACKAGE_ARCH=arm
	make package-rpm-fpm ARCH=arm PACKAGE_ARCH=armhf

package-deps:
	# Installing packaging dependencies...
	gem install fpm

package-deb-fpm:
	@mkdir -p out/deb/
	fpm -s dir -t deb -n $(PACKAGE_NAME) -v $(VERSION) \
		-p out/deb/$(PACKAGE_NAME)_$(PACKAGE_ARCH).deb \
		--deb-priority optional --category admin \
		--force \
		--deb-compression bzip2 \
		--after-install packaging/scripts/postinst.deb \
		--before-remove packaging/scripts/prerm.deb \
		--url https://gitlab.com/gitlab-org/gitlab-ci-multi-runner \
		--description "GitLab Runner" \
		-m "Kamil Trzciński <ayufan@ayufan.eu>" \
		--license "MIT" \
		--vendor "ayufan.eu" \
		--conflicts $(PACKAGE_CONFLICT) \
		-a $(PACKAGE_ARCH) \
		out/binaries/$(NAME)-linux-$(ARCH)=/usr/bin/gitlab-ci-multi-runner

package-rpm-fpm:
	@mkdir -p out/rpm/
	fpm -s dir -t rpm -n $(PACKAGE_NAME) -v $(VERSION) \
		-p out/rpm/$(PACKAGE_NAME)_$(PACKAGE_ARCH).rpm \
		--rpm-compression bzip2 --rpm-os linux \
		--force \
		--after-install packaging/scripts/postinst.rpm \
		--before-remove packaging/scripts/prerm.rpm \
		--url https://gitlab.com/gitlab-org/gitlab-ci-multi-runner \
		--description "GitLab Runner" \
		-m "Kamil Trzciński <ayufan@ayufan.eu>" \
		--license "MIT" \
		--vendor "ayufan.eu" \
		--conflicts $(PACKAGE_CONFLICT) \
		-a $(PACKAGE_ARCH) \
		out/binaries/$(NAME)-linux-$(ARCH)=/usr/bin/gitlab-ci-multi-runner

packagecloud: packagecloud-deps packagecloud-deb packagecloud-rpm

packagecloud-deps:
	# Installing packagecloud dependencies...
	gem install package_cloud

packagecloud-deb:
	# Sending Debian compatible packages...
	-for DIST in $(DEB_PLATFORMS); do \
		package_cloud push --url $(PACKAGE_CLOUD_URL) $(PACKAGE_CLOUD)/$$DIST out/deb/*.deb; \
	done

packagecloud-rpm:
	# Sending RedHat compatible packages...
	-for DIST in $(RPM_PLATFORMS); do \
		package_cloud push --url $(PACKAGE_CLOUD_URL) $(PACKAGE_CLOUD)/$$DIST out/rpm/*.rpm; \
	done

packagecloud-yank:
ifneq ($(YANK),)
	# Removing $(YANK) from packagecloud...
	-for DIST in $(DEB_PLATFORMS); do \
		for ARCH in $(DEB_ARCHS); do \
			package_cloud yank --url $(PACKAGE_CLOUD_URL) $(PACKAGE_CLOUD)/$$DIST $(PACKAGE_NAME)_$(YANK)_$$ARCH.deb & \
		done; \
	done; \
	for DIST in $(RPM_PLATFORMS); do \
		for ARCH in $(RPM_ARCHS); do \
			package_cloud yank --url $(PACKAGE_CLOUD_URL) $(PACKAGE_CLOUD)/$$DIST $(PACKAGE_NAME)-$(YANK)-1.$$ARCH.rpm & \
		done; \
	done; \
	wait
else
	# No version specified in YANK
	@exit 1
endif

s3-upload:
	export ARTIFACTS_DEST=artifacts; curl -sL https://raw.githubusercontent.com/travis-ci/artifacts/master/install | bash
	./artifacts upload \
		--permissions public-read \
		--working-dir out \
		--target-paths "$(S3_UPLOAD_PATH)/" \
		$(shell cd out/; find . -type f)

FORCE:
