package common

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ayufan/gitlab-ci-multi-runner/helpers"
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
	Runner        *RunnerConfig  `json:"runner"`

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

func (b *Build) writeCloneCmd(w io.Writer, buildsDir string) {
	io.WriteString(w, "echo Clonning repository...\n")
	io.WriteString(w, fmt.Sprintf("mkdir -p %s\n", buildsDir))
	io.WriteString(w, fmt.Sprintf("cd %s\n", buildsDir))
	io.WriteString(w, fmt.Sprintf("rm -rf %s\n", b.ProjectDir()))
	io.WriteString(w, fmt.Sprintf("git clone %s %s\n", b.RepoURL, b.ProjectDir()))
	io.WriteString(w, fmt.Sprintf("cd %s\n", b.ProjectDir()))
}

func (b *Build) writeFetchCmd(w io.Writer, buildsDir string) {
	io.WriteString(w, fmt.Sprintf("if [[ -d %s/%s/.git ]]; then\n", buildsDir, b.ProjectDir()))
	io.WriteString(w, "echo Fetching changes...\n")
	io.WriteString(w, fmt.Sprintf("cd %s/%s\n", buildsDir, b.ProjectDir()))
	io.WriteString(w, fmt.Sprintf("git clean -fdx\n"))
	io.WriteString(w, fmt.Sprintf("git reset --hard > /dev/null\n"))
	io.WriteString(w, fmt.Sprintf("git remote set-url origin %s\n", b.RepoURL))
	io.WriteString(w, fmt.Sprintf("git fetch origin\n"))
	io.WriteString(w, fmt.Sprintf("else\n"))
	b.writeCloneCmd(w, buildsDir)
	io.WriteString(w, fmt.Sprintf("fi\n"))
}

func (b *Build) writeCheckoutCmd(w io.Writer, buildsDir string) {
	io.WriteString(w, fmt.Sprintf("echo Checkouting %s as %s...\n", b.Sha[0:8], b.RefName))
	io.WriteString(w, fmt.Sprintf("git checkout -B %s %s > /dev/null\n", b.RefName, b.Sha))
	io.WriteString(w, fmt.Sprintf("git reset --hard %s > /dev/null\n", b.Sha))
}

func (b *Build) Generate(buildsDir string, hostname string) ([]byte, []string, error) {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)

	io.WriteString(w, "#!/usr/bin/env bash\n")
	io.WriteString(w, "\n")
	if len(hostname) != 0 {
		io.WriteString(w, fmt.Sprintf("echo Running on $(hostname) via %s...\n", helpers.ShellEscape(hostname)))
	} else {
		io.WriteString(w, "echo Running on $(hostname)...\n")
	}
	io.WriteString(w, "\n")
	io.WriteString(w, "trap 'kill -s INT 0' EXIT\n")
	io.WriteString(w, "set -eo pipefail\n")

	io.WriteString(w, "\n")
	if b.AllowGitFetch {
		b.writeFetchCmd(w, buildsDir)
	} else {
		b.writeCloneCmd(w, buildsDir)
	}

	b.writeCheckoutCmd(w, buildsDir)
	io.WriteString(w, "\n")
	if !b.Runner.DisableVerbose {
		io.WriteString(w, "set -v\n")
		io.WriteString(w, "\n")
	}

	commands := b.Commands
	commands = strings.Replace(commands, "\r\n", "\n", -1)

	io.WriteString(w, commands)

	w.Flush()

	env := []string{
		fmt.Sprintf("CI_BUILD_REF=%s", b.Sha),
		fmt.Sprintf("CI_BUILD_BEFORE_SHA=%s", b.BeforeSha),
		fmt.Sprintf("CI_BUILD_REF_NAME=%s", b.RefName),
		fmt.Sprintf("CI_BUILD_ID=%d", b.ID),
		fmt.Sprintf("CI_BUILD_REPO=%s", b.RepoURL),

		fmt.Sprintf("CI_PROJECT_ID=%d", b.ProjectID),
		fmt.Sprintf("CI_PROJECT_DIR=%s", buildsDir, b.ProjectDir()),

		"CI_SERVER=yes",
		"CI_SERVER_NAME=GitLab CI",
		"CI_SERVER_VERSION=",
		"CI_SERVER_REVISION=",
	}

	return buffer.Bytes(), env, nil
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
