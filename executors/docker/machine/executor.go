package machine

import (
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"

	// Force to load docker executor
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/docker"
)

type machineExecutor struct {
	provider      *machineProvider
	otherExecutor common.Executor
	data          common.ExecutorData
	config        common.RunnerConfig
}

func (e *machineExecutor) Prepare(globalConfig *common.Config, config *common.RunnerConfig, build *common.Build) (err error) {
	// Use the machine
	e.config, e.data, err = e.provider.Use(config, build.ExecutorData)
	if err != nil {
		return err
	}

	// TODO: Currently the docker-machine doesn't support multiple builds
	build.ProjectRunnerID = 0
	if details, _ := build.ExecutorData.(*machineDetails); details != nil {
		build.Hostname = details.Name
	} else if details, _ := e.data.(*machineDetails); details != nil {
		build.Hostname = details.Name
	}

	return e.otherExecutor.Prepare(globalConfig, &e.config, build)
}

func (e *machineExecutor) Start() error {
	return e.otherExecutor.Start()
}

func (e *machineExecutor) Wait() error {
	return e.otherExecutor.Wait()
}

func (e *machineExecutor) Finish(err error) {
	e.otherExecutor.Finish(err)
}

func (e *machineExecutor) Cleanup() {
	e.otherExecutor.Cleanup()

	// Release allocated machine
	if e.data != "" {
		e.provider.Release(&e.config, e.data)
		e.data = nil
	}
}

func init() {
	common.RegisterExecutor("docker+machine", newMachineProvider("docker"))
	common.RegisterExecutor("docker-ssh+machine", newMachineProvider("docker-ssh"))
}
