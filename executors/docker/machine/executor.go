package machine

import (
	"errors"

	"github.com/Sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"

	// Force to load docker executor
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/docker"
)

type machineExecutor struct {
	provider *machineProvider
	executor common.Executor
	build    *common.Build
	data     common.ExecutorData
	config   common.RunnerConfig
}

func (e *machineExecutor) log() (log *logrus.Entry) {
	log = e.build.Log()

	details, _ := e.build.ExecutorData.(*machineDetails)
	if details == nil {
		details, _ = e.data.(*machineDetails)
	}
	if details != nil {
		log = log.WithFields(logrus.Fields{
			"name":      details.Name,
			"usedcount": details.UsedCount,
			"created":   details.Created,
		})
	}
	if e.config.Docker != nil {
		log = log.WithField("docker", e.config.Docker.Host)
	}

	return
}

func (e *machineExecutor) Shell() *common.ShellScriptInfo {
	if e.executor == nil {
		return nil
	}
	return e.executor.Shell()
}

func (e *machineExecutor) Prepare(globalConfig *common.Config, config *common.RunnerConfig, build *common.Build) (err error) {
	e.build = build

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

	e.log().Infoln("Starting docker-machine build...")

	// Create original executor
	e.executor = e.provider.provider.Create()
	if e.executor == nil {
		return errors.New("failed to create an executor")
	}
	return e.executor.Prepare(globalConfig, &e.config, build)
}

func (e *machineExecutor) Run(cmd common.ExecutorCommand) error {
	if e.executor == nil {
		return errors.New("missing executor")
	}
	return e.executor.Run(cmd)
}

func (e *machineExecutor) Finish(err error) {
	if e.executor != nil {
		e.executor.Finish(err)
	}
	e.log().Infoln("Finished docker-machine build:", err)
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
