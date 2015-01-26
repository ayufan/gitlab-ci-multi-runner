all:
	GOOS=linux GOARCH=386 bash -c 'go build -o out/gitlab-ci-multirunner-$$GOOS-$$GOARCH$$GOEXT'
	GOOS=linux GOARCH=amd64 bash -c 'go build -o out/gitlab-ci-multirunner-$$GOOS-$$GOARCH$$GOEXT'
	GOOS=darwin GOARCH=amd64 bash -c 'go build -o out/gitlab-ci-multirunner-$$GOOS-$$GOARCH$$GOEXT'
	GOOS=windows GOARCH=amd64 GOEXT=.exe bash -c 'go build -o out/gitlab-ci-multirunner-$$GOOS-$$GOARCH$$GOEXT'
