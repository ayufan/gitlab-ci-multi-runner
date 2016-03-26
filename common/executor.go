package common

import (
	log "github.com/Sirupsen/logrus"
)

type ExecutorData interface{}

const (
	ImageDefault        string = ""
	ImageInternalPrefix        = "***"
	ImagePreBuild              = ImageInternalPrefix + "custom-pre-build-image"
	ImagePostBuild             = ImageInternalPrefix + "custom-post-build-image"
)

type ExecutorRun struct {
	ShellScript

	Image string
	Trace BuildTrace
	Abort chan error
}

type Executor interface {
	Prepare(build *Build, data ExecutorData) error
	Run(run ExecutorRun) error
	Cleanup()
}

type ExecutorProvider interface {
	CanCreate() bool
	Create() Executor
	Acquire(config *RunnerConfig) (ExecutorData, error)
	Release(config *RunnerConfig, data ExecutorData) error
	GetFeatures(features *FeaturesInfo)
}

var executors map[string]ExecutorProvider

func RegisterExecutor(executor string, provider ExecutorProvider) {
	log.Debugln("Registering", executor, "executor...")

	if executors == nil {
		executors = make(map[string]ExecutorProvider)
	}
	if _, ok := executors[executor]; ok {
		panic("Executor already exist: " + executor)
	}
	executors[executor] = provider
}

func GetExecutor(executor string) ExecutorProvider {
	if executors == nil {
		return nil
	}

	provider, _ := executors[executor]
	return provider
}

func GetExecutors() []string {
	names := []string{}
	if executors != nil {
		for name := range executors {
			names = append(names, name)
		}
	}
	return names
}

func NewExecutor(executor string) Executor {
	provider := GetExecutor(executor)
	if provider != nil {
		return provider.Create()
	}

	return nil
}
