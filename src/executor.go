package src

import (
)

type Executor interface {
	Run(config RunnerConfig, build Build) error
}

func GetExecutor(config RunnerConfig) Executor {
	switch config.Executor {
		case "shell":
			return &ShellExecutor{}
		case "":
			return &ShellExecutor{}
		default:
			return nil
	}
}
