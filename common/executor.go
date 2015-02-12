package common

type Executor interface {
	Prepare(config *RunnerConfig, build *Build) error
	Start() error
	Wait() error
	Cleanup()
}

var executors map[string]func() Executor

func RegisterExecutor(executor string, closure func() Executor) {
	if executors == nil {
		executors = make(map[string]func() Executor)
	}
	if executors[executor] != nil {
		panic("Executor already exist: " + executor)
	}
	executors[executor] = closure
}

func GetExecutor(executor string) Executor {
	if executors == nil {
		return nil
	}

	closure := executors[executor]
	if closure == nil {
		return nil
	}

	return closure()
}
