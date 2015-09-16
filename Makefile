NAME ?= gitlab-ci-multi-runner
PACKAGE_NAME ?= $(NAME)
PACKAGE_CONFLICT ?= $(PACKAGE_NAME)-beta
REVISION := $(shell git rev-parse --short HEAD || echo unknown)
VERSION := $(shell git describe --tags || cat VERSION || echo dev)
VERSION := $(shell echo $(VERSION) | sed -e 's/^v//g')
ifneq ($(RELEASE),true)
    VERSION := $(shell echo $(VERSION)-beta)
endif
ITTERATION := $(shell date +%s)
PACKAGE_CLOUD ?= ayufan/gitlab-ci-multi-runner
PACKAGE_CLOUD_URL ?= https://packagecloud.io/
BUILD_PLATFORMS ?= -os="linux" -os="darwin" -os="windows"
S3_UPLOAD_PATH ?= master
DEB_PLATFORMS ?= debian/wheezy debian/jessie ubuntu/precise ubuntu/trusty ubuntu/utopic ubuntu/vivid
DEB_ARCHS ?= amd64 i386 arm armhf
RPM_PLATFORMS ?= el/6 el/7 ol/6 ol/7
RPM_ARCHS ?= x86_64 i686 arm armhf

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
	-go get golang.org/x/sys/windows/svc
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
	go test ./... -cover

test-docker:
	make test-docker-image IMAGE=centos:6 TYPE=rpm
	make test-docker-image IMAGE=centos:7 TYPE=rpm
	make test-docker-image IMAGE=debian:wheezy TYPE=deb
	make test-docker-image IMAGE=debian:jessie TYPE=deb
	make test-docker-image IMAGE=ubuntu-upstart:precise TYPE=deb
	make test-docker-image IMAGE=ubuntu-upstart:trusty TYPE=deb
	make test-docker-image IMAGE=ubuntu-upstart:utopic TYPE=deb

test-docker-image:
	tests/test_installation.sh $(IMAGE) out/$(TYPE)/$(PACKAGE_NAME)_amd64.$(TYPE)
	tests/test_installation.sh $(IMAGE) out/$(TYPE)/$(PACKAGE_NAME)_amd64.$(TYPE) Y

build-and-deploy:
	make build BUILD_PLATFORMS="-os=linux -arch=amd64"
	make package-deb-fpm ARCH=amd64 PACKAGE_ARCH=amd64
	scp out/deb/$(PACKAGE_NAME)_amd64.deb $(SERVER):
	ssh $(SERVER) dpkg -i $(PACKAGE_NAME)_amd64.deb

version: FORCE
	# Generating VERSION...
	echo "package common\n\nconst NAME = \"$(PACKAGE_NAME)\"\nconst VERSION = \"$(VERSION)\"\nconst REVISION = \"$(REVISION)\"" > common/version.go

package: package-deps package-deb package-rpm

package-deb:
	# Building Debian compatible packages...
	make package-deb-fpm ARCH=amd64 PACKAGE_ARCH=amd64
	make package-deb-fpm ARCH=386 PACKAGE_ARCH=i386
	make package-deb-fpm ARCH=arm PACKAGE_ARCH=arm
	make package-deb-fpm ARCH=arm PACKAGE_ARCH=armhf

package-rpm:
	# Building RedHat compatible packages...
	make package-rpm-fpm ARCH=amd64 PACKAGE_ARCH=amd64
	make package-rpm-fpm ARCH=386 PACKAGE_ARCH=i686
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
		--provides gitlab-runner \
		--replaces gitlab-runner \
		--depends ca-certificates \
		--depends git \
		--deb-suggests docker \
		-a $(PACKAGE_ARCH) \
		packaging/root/=/ \
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
		--provides gitlab-runner \
		--replaces gitlab-runner \
		-a $(PACKAGE_ARCH) \
		packaging/root/=/ \
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
