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
