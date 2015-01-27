all: linux-386 linux darwin-386 darwin

linux-386:
	GOOS=linux GOARCH=386 bash -c 'go build -o out/gitlab-ci-multi-runner-$$GOOS-$$GOARCH$$GOEXT'

linux:
	GOOS=linux GOARCH=amd64 bash -c 'go build -o out/gitlab-ci-multi-runner-$$GOOS-$$GOARCH$$GOEXT'

darwin-386:
	GOOS=darwin GOARCH=386 bash -c 'go build -o out/gitlab-ci-multi-runner-$$GOOS-$$GOARCH$$GOEXT'

darwin:
	GOOS=darwin GOARCH=amd64 bash -c 'go build -o out/gitlab-ci-multi-runner-$$GOOS-$$GOARCH$$GOEXT'

windows-386:
	GOOS=windows GOARCH=386 GOEXT=.exe bash -c 'go build -o out/gitlab-ci-multi-runner-$$GOOS-$$GOARCH$$GOEXT'

windows:
	GOOS=windows GOARCH=amd64 GOEXT=.exe bash -c 'go build -o out/gitlab-ci-multi-runner-$$GOOS-$$GOARCH$$GOEXT'
