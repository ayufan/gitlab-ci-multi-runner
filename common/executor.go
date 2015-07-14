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

var executors map[string]func() Executor

func RegisterExecutor(executor string, closure func() Executor) {
	log.Debugln("Registering", executor, "executor...")

	if executors == nil {
		executors = make(map[string]func() Executor)
	}
	if executors[executor] != nil {
		panic("Executor already exist: " + executor)
	}
	executors[executor] = closure
}

func NewExecutor(executor string) Executor {
	if executors == nil {
		return nil
	}

	closure := executors[executor]
	if closure == nil {
		return nil
	}

	return closure()
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
