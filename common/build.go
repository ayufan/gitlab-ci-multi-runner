package common

import (
	"errors"
	"fmt"
	"os"
	"time"
)

type BuildState string

const (
	Pending BuildState = "pending"
	Running            = "running"
	Failed             = "failed"
	Success            = "success"
)

type Build struct {
	GetBuildResponse
	BuildLog      string         `json:"-"`
	BuildState    BuildState     `json:"build_state"`
	BuildStarted  time.Time      `json:"build_started"`
	BuildFinished time.Time      `json:"build_finished"`
	BuildDuration time.Duration  `json:"build_duration"`
	BuildMessage  string         `json:"build_message"`
	BuildAbort    chan os.Signal `json:"-"`
	BuildsDir     string
	Hostname      string
	Runner        *RunnerConfig `json:"runner"`

	GlobalID   int    `json:"global_id"`
	GlobalName string `json:"global_name"`

	RunnerID   int    `json:"runner_id"`
	RunnerName string `json:"runner_name"`

	ProjectRunnerID   int    `json:"project_runner_id"`
	ProjectRunnerName string `json:"name"`
}

func (b *Build) Prepare(otherBuilds []*Build) {
	globals := make(map[int]bool)
	runners := make(map[int]bool)
	projectRunners := make(map[int]bool)

	for _, otherBuild := range otherBuilds {
		globals[otherBuild.GlobalID] = true

		if otherBuild.Runner.ShortDescription() != b.Runner.ShortDescription() {
			continue
		}
		runners[otherBuild.RunnerID] = true

		if otherBuild.ProjectID != b.ProjectID {
			continue
		}
		projectRunners[otherBuild.ProjectRunnerID] = true
	}

	for i := 0; ; i++ {
		if !globals[i] {
			b.GlobalID = i
			b.GlobalName = fmt.Sprintf("concurrent-%d", i)
			break
		}
	}

	for i := 0; ; i++ {
		if !runners[i] {
			b.RunnerID = i
			b.RunnerName = fmt.Sprintf("runner-%s-concurrent-%d",
				b.Runner.ShortDescription(), i)
			break
		}
	}

	for i := 0; ; i++ {
		if !projectRunners[i] {
			b.ProjectRunnerID = i
			b.ProjectRunnerName = fmt.Sprintf("runner-%s-project-%d-concurrent-%d",
				b.Runner.ShortDescription(), b.ProjectID, i)
			break
		}
	}

	b.BuildAbort = make(chan os.Signal, 1)
}

func (b *Build) ProjectUniqueName() string {
	return b.ProjectRunnerName
}

func (b *Build) ProjectDir() string {
	return b.ProjectUniqueName()
}

func (b *Build) FullProjectDir() string {
	return fmt.Sprintf("%s/%s", b.BuildsDir, b.ProjectDir())
}

func (b *Build) Run() error {
	var err error
	executor := GetExecutor(b.Runner.Executor)
	if executor == nil {
		err = errors.New("executor not found")
	}
	if err == nil {
		err = executor.Prepare(b.Runner, b)
	}
	if err == nil {
		err = executor.Start()
	}
	if err == nil {
		err = executor.Wait()
	}
	executor.Finish(err)
	if executor != nil {
		executor.Cleanup()
	}
	return err
}
