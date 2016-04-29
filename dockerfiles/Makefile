all: cache service

help:
	# make alpine
	# make cache
	# make dind
	# make service
	# make ubuntu

alpine: FORCE
	docker build -t gitlab/gitlab-runner:alpine alpine/

cache: FORCE
	docker build -t gitlab/gitlab-runner:cache cache/

dind: FORCE
	docker build -t gitlab/gitlab-runner:dind dind/

service: FORCE
	docker build -t gitlab/gitlab-runner:service service/

ubuntu: FORCE
	docker build -t gitlab/gitlab-runner:ubuntu ubuntu/
	docker tag gitlab/gitlab-runner:ubuntu  gitlab/gitlab-runner:latest

FORCE:
