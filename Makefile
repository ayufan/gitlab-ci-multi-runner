all: build

build:
	gox -os="linux" -os="darwin" -output="out/{{.Dir}}-{{.OS}}-{{.Arch}}"

test:
	go test

deploy:
	gox -osarch="linux/amd64" -output="out/{{.Dir}}-{{.OS}}-{{.Arch}}"
	scp out/gitlab-ci-multi-runner-linux-amd64 lab-worker:
	ssh lab-worker ./gitlab-ci-multi-runner-linux-amd64 --debug run
