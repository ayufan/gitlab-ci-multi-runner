package common

import (
	log "github.com/Sirupsen/logrus"
)

type Executor interface {
	Prepare(globalConfig *Config, config *RunnerConfig, build *Build) error
	Start() error
	Wait() error
	Finish(err error)
	Cleanup()
}

type ExecutorFactory struct {
	Create   func() Executor
	Features FeaturesInfo
}

var executors map[string]ExecutorFactory

func RegisterExecutor(executor string, factory ExecutorFactory) {
	log.Debugln("Registering", executor, "executor...")

	if executors == nil {
		executors = make(map[string]ExecutorFactory)
	}
	if _, ok := executors[executor]; ok {
		panic("Executor already exist: " + executor)
	}
	executors[executor] = factory
}

func GetExecutorFeatures(executor string) *FeaturesInfo {
	if executors == nil {
		return nil
	}

	if factory, ok := executors[executor]; ok {
		return &factory.Features
	}

	return nil
}

func NewExecutor(executor string) Executor {
	if executors == nil {
		return nil
	}

	if factory, ok := executors[executor]; ok {
		return factory.Create()
	}

	return nil
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
