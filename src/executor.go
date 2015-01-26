package src

type Executor interface {
	Prepare(config *RunnerConfig, build *Build) error
	Start() error
	Wait() error
	Cleanup()
}

func GetExecutor(config RunnerConfig) Executor {
	switch config.Executor {
	case "shell", "":
		return &ShellExecutor{
			AbstractExecutor: AbstractExecutor{
				DefaultBuildsDir: "tmp/builds",
			},
		}
	case "docker":
		return &DockerCommandExecutor{
			DockerExecutor: DockerExecutor{
				AbstractExecutor: AbstractExecutor{
					DefaultBuildsDir: "/builds",
				},
			},
		}
	case "docker-ssh":
		return &DockerSshExecutor{
			DockerExecutor: DockerExecutor{
				AbstractExecutor: AbstractExecutor{
					DefaultBuildsDir: "builds",
				},
			},
		}
	default:
		return nil
	}
}
