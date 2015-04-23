package common

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
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
	BuildState    BuildState     `json:"build_state"`
	BuildStarted  time.Time      `json:"build_started"`
	BuildFinished time.Time      `json:"build_finished"`
	BuildDuration time.Duration  `json:"build_duration"`
	BuildMessage  string         `json:"build_message"`
	BuildAbort    chan os.Signal `json:"-"`
	BuildDir      string
	Hostname      string
	Runner        *RunnerConfig `json:"runner"`

	GlobalID   int    `json:"global_id"`
	GlobalName string `json:"global_name"`

	RunnerID   int    `json:"runner_id"`
	RunnerName string `json:"runner_name"`

	ProjectRunnerID   int    `json:"project_runner_id"`
	ProjectRunnerName string `json:"name"`

	buildLog     bytes.Buffer `json:"-"`
	buildLogLock sync.RWMutex
}

func (b *Build) AssignID(otherBuilds ...*Build) {
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
}

func (b *Build) ProjectUniqueName() string {
	return b.ProjectRunnerName
}

func (b *Build) ProjectSlug() (string, error) {
	splits := strings.Split(b.RepoURL, "/")
	if len(splits) <= 2 {
		return "", errors.New("invalid URL")
	}

	slug := strings.Join(splits[len(splits)-2:], "/")
	slug = strings.TrimSuffix(slug, ".git")
	return slug, nil
}

func (b *Build) FullProjectDir() string {
	return b.BuildDir
}

func (b *Build) StartBuild(buildDir string) {
	b.BuildStarted = time.Now()
	b.BuildState = Pending
	b.BuildDir = buildDir
}

func (b *Build) FinishBuild(buildState BuildState, buildMessage string, args ...interface{}) {
	b.BuildState = buildState
	b.BuildMessage = "\n" + fmt.Sprintf(buildMessage, args...)
	b.BuildFinished = time.Now()
	b.BuildDuration = b.BuildFinished.Sub(b.BuildStarted)
}

func (b *Build) BuildLog() string {
	b.buildLogLock.RLock()
	defer b.buildLogLock.RUnlock()
	return b.buildLog.String()
}

func (b *Build) BuildLogLen() int {
	b.buildLogLock.RLock()
	defer b.buildLogLock.RUnlock()
	return b.buildLog.Len()
}

func (b *Build) WriteString(data string) (int, error) {
	b.buildLogLock.Lock()
	defer b.buildLogLock.Unlock()
	return b.buildLog.WriteString(data)
}

func (b *Build) WriteRune(r rune) (int, error) {
	b.buildLogLock.Lock()
	defer b.buildLogLock.Unlock()
	return b.buildLog.WriteRune(r)
}

func (b *Build) SendBuildLog() {
	var buildTrace string

	buildTrace = b.BuildLog()
	if b.BuildMessage != "" {
		buildTrace = buildTrace + b.BuildMessage
	}

	for {
		if UpdateBuild(*b.Runner, b.ID, b.BuildState, buildTrace) != UpdateFailed {
			break
		} else {
			time.Sleep(UpdateRetryInterval * time.Second)
		}
	}
}

func (b *Build) Run() error {
	executor := GetExecutor(b.Runner.Executor)
	if executor == nil {
		b.FinishBuild(Failed, "Executor not found: %v", b.Runner.Executor)
		b.SendBuildLog()
		return errors.New("executor not found")
	}

	err := executor.Prepare(b.Runner, b)
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
