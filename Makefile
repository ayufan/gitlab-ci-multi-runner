NAME ?= gitlab-ci-multi-runner
PACKAGE_NAME ?= $(NAME)
PACKAGE_CONFLICT ?= $(PACKAGE_NAME)-beta
REVISION := $(shell git rev-parse --short HEAD || echo unknown)
LAST_TAG := $(shell git describe --tags --abbrev=0)
COMMITS := $(shell echo `git log --oneline $(LAST_TAG)..HEAD | wc -l`)
VERSION := $(shell (cat VERSION || echo dev) | sed -e 's/^v//g')
ifneq ($(RELEASE),true)
    VERSION := $(shell echo $(VERSION)~beta.$(COMMITS).g$(REVISION))
endif
ITTERATION := $(shell date +%s)
PACKAGE_CLOUD ?= ayufan/gitlab-ci-multi-runner
PACKAGE_CLOUD_URL ?= https://packagecloud.io/
BUILD_PLATFORMS ?= -os '!netbsd' -os '!openbsd'
S3_UPLOAD_PATH ?= master
DEB_PLATFORMS ?= debian/wheezy debian/jessie debian/stretch debian/buster \
    ubuntu/precise ubuntu/trusty ubuntu/utopic ubuntu/vivid ubuntu/wily ubuntu/xenial \
    raspbian/wheezy raspbian/jessie raspbian/stretch raspbian/buster \
    linuxmint/petra linuxmint/qiana linuxmint/rebecca linuxmint/rafaela linuxmint/rosa
DEB_ARCHS ?= amd64 i386 arm armhf
RPM_PLATFORMS ?= el/6 el/7 \
    ol/6 ol/7 \
    fedora/20 fedora/21 fedora/22 fedora/23
RPM_ARCHS ?= x86_64 i686 arm armhf
GO_LDFLAGS ?= -X main.NAME $(PACKAGE_NAME) -X main.VERSION $(VERSION) -X main.REVISION $(REVISION)
GO_FILES ?= $(shell find . -name '*.go')
export GO15VENDOREXPERIMENT := 1
export CGO_ENABLED := 0

all: deps verify build

help:
	# Commands:
	# make all => deps verify toolchain build
	# make version - show information about current version
	#
	# Development commands:
	# make install - install the version suitable for your OS as gitlab-ci-multi-runner
	# make docker - build docker dependencies
	#
	# Testing commands:
	# make verify - run fmt, complexity, test and lint
	# make fmt - check source formatting
	# make test - run project tests
	# make lint - check project code style
	# make vet - examine code and report suspicious constructs
	# make complexity - check code complexity
	#
	# Deployment commands:
	# make deps - install all dependencies
	# make build - build project for all supported OSes
	# make package - package project using FPM
	# make packagecloud - send all packages to packagecloud
	# make packagecloud-yank - remove specific version from packagecloud

version: FORCE
	@echo Current version: $(VERSION)
	@echo Current iteration: $(ITTERATION)
	@echo Current revision: $(REVISION)
	@echo DEB platforms: $(DEB_PLATFORMS)
	@echo RPM platforms: $(RPM_PLATFORMS)
	bash -c 'echo TEST: ${GO15VENDOREXPERIMENT}'

verify: fmt vet lint complexity test

deps:
	# Installing dependencies...
	go get -u github.com/golang/lint/golint
	go get github.com/mitchellh/gox
	go get golang.org/x/tools/cmd/cover
	go get github.com/fzipp/gocyclo
	go get -u github.com/jteeuwen/go-bindata/...
	go install cmd/vet

out/docker/prebuilt.tar.gz: $(GO_FILES)
	# Create directory
	mkdir -p out/docker

ifneq (, $(shell docker info))
	# Building gitlab-runner-helper
	gox -osarch=linux/amd64 \
		-ldflags "$(GO_LDFLAGS)" \
		-output="dockerfiles/build/gitlab-runner-helper" \
		./apps/gitlab-runner-helper

	# Build docker images
	docker build -t gitlab-runner-build:$(REVISION) dockerfiles/build
	docker build -t gitlab-runner-cache:$(REVISION) dockerfiles/cache
	docker build -t gitlab-runner-service:$(REVISION) dockerfiles/service
	docker save -o out/docker/prebuilt.tar \
		gitlab-runner-build:$(REVISION) \
		gitlab-runner-service:$(REVISION) \
		gitlab-runner-cache:$(REVISION)
	gzip -f -9 out/docker/prebuilt.tar
else
	$(warning =============================================)
	$(warning WARNING: downloading prebuilt docker images that will be embedded in gitlab-runner)
	$(warning WARNING: to use images compiled from your code install Docker Engine)
	$(warning WARNING: and remove out/docker/prebuilt.tar.gz)
	$(warning =============================================)
	curl -o out/docker/prebuilt.tar.gz \
		https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/master/docker/prebuilt.tar.gz
endif

executors/docker/bindata.go: out/docker/prebuilt.tar.gz
	# Generating embedded data
	go-bindata \
		-pkg docker \
		-nocompress \
		-nomemcopy \
		-nometadata \
		-prefix out/docker/ \
		-o executors/docker/bindata.go \
		out/docker/prebuilt.tar.gz

docker: executors/docker/bindata.go

build: executors/docker/bindata.go
	# Building gitlab-ci-multi-runner for $(BUILD_PLATFORMS)
	gox $(BUILD_PLATFORMS) \
		-ldflags "$(GO_LDFLAGS)" \
		-output="out/binaries/$(NAME)-{{.OS}}-{{.Arch}}"

fmt:
	# Checking project code formatting...
	@go fmt ./... | awk '{ print "Please run go fmt"; exit 1 }'

vet:
	# Checking for suspicious constructs...
	@go vet ./...

lint:
	# Checking project code style...
	@golint ./... | ( ! grep -v -e "be unexported" -e "don't use an underscore in package name" -e "ALL_CAPS" )

complexity:
	# Checking code complexity
	@gocyclo -over 9 $(shell find . -name '*.go' | grep -v \
	    -e "/Godeps" \
	    -e "/helpers/shell_escape.go" \
	    -e "/executors/parallels/" \
	    -e "/executors/virtualbox/")

test: executors/docker/bindata.go
	# Running tests...
	@go test ./... -cover

install: executors/docker/bindata.go
	go install --ldflags="$(GO_LDFLAGS)"

dockerfiles:
	make -C dockerfiles all

mocks: FORCE
	go get github.com/vektra/mockery/.../
	rm -rf mocks/
	mockery -dir=$(GOPATH)/src/github.com/ayufan/golang-kardianos-service -name=Interface
	mockery -dir=./common -name=Network
	mockery -dir=./helpers/docker -name=Client

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

build-and-deploy-binary:
	make build BUILD_PLATFORMS="-os=linux -arch=amd64"
	scp out/binaries/$(PACKAGE_NAME)-linux-amd64 $(SERVER):/usr/bin/gitlab-runner

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
		--depends curl \
		--depends tar \
		--deb-suggests docker-engine \
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
		--depends git \
		--depends curl \
		--depends tar \
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
