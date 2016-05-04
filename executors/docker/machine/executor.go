package machine

import (
	"errors"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"

	// Force to load docker executor
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/docker"
)

type machineExecutor struct {
	provider *machineProvider
	executor common.Executor
	data     common.ExecutorData
	config   common.RunnerConfig
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

	// Create original executor
	e.executor = e.provider.provider.Create()
	if e.executor == nil {
		return errors.New("failed to create an executor")
	}
	return e.executor.Prepare(globalConfig, &e.config, build)
}

func (e *machineExecutor) Start() error {
	if e.executor == nil {
		return errors.New("missing executor")
	}
	return e.executor.Start()
}

func (e *machineExecutor) Run(cmd common.ExecutorCommand) error {
	if e.executor == nil {
		return errors.New("missing executor")
	}
	return e.executor.Run(cmd)
}

func (e *machineExecutor) Wait() error {
	if e.executor == nil {
		return errors.New("missing executor")
	}
	return e.executor.Wait()
}

func (e *machineExecutor) Finish(err error) {
	if e.executor != nil {
		e.executor.Finish(err)
	}
}

func (e *machineExecutor) Cleanup() {
	// Cleanup executor if were created
	if e.executor != nil {
		e.executor.Cleanup()
	}

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
