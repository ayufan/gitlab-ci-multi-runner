package src

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
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
	Name          string        `json:"name"`
	BuildLog      string        `json:"-"`
	BuildState    BuildState    `json:"build_state"`
	BuildStarted  time.Time     `json:"build_started"`
	BuildFinished time.Time     `json:"build_finished"`
	BuildDuration time.Duration `json:"build_duration"`
	Runner        *RunnerConfig `json:"runner"`
}

func (b *Build) isNameUnique(other_builds []*Build) bool {
	for _, other_build := range other_builds {
		if b.Name == other_build.Name {
			return false
		}
	}
	return true
}

func (b *Build) GenerateUniqueName(prefix string, other_builds []*Build) {
	for i := 0; ; i++ {
		b.Name = fmt.Sprintf("%s-project-%d-%d", prefix, b.ProjectId, i)
		if b.isNameUnique(other_builds) {
			return
		}
	}
}

func (b *Build) ProjectUniqueName() string {
	if len(b.Name) == 0 {
		return fmt.Sprintf("project-%d", b.ProjectId)
	} else {
		return b.Name
	}
}

func (b *Build) ProjectDir() string {
	return b.ProjectUniqueName()
}

func (b *Build) writeCloneCmd(w io.Writer, builds_dir string) {
	io.WriteString(w, "echo Clonning repository...\n")
	io.WriteString(w, fmt.Sprintf("mkdir -p %s && ", builds_dir))
	io.WriteString(w, fmt.Sprintf("cd %s && ", builds_dir))
	io.WriteString(w, fmt.Sprintf("rm -rf %s && ", b.ProjectDir()))
	io.WriteString(w, fmt.Sprintf("git clone %s %s && ", b.RepoURL, b.ProjectDir()))
	io.WriteString(w, fmt.Sprintf("cd %s\n", b.ProjectDir()))
}

func (b *Build) writeFetchCmd(w io.Writer, builds_dir string) {
	io.WriteString(w, fmt.Sprintf("if [[ -d %s/%s/.git ]]; then\n", builds_dir, b.ProjectDir()))
	io.WriteString(w, "echo Fetching changes...\n")
	io.WriteString(w, fmt.Sprintf("cd %s/%s && ", builds_dir, b.ProjectDir()))
	io.WriteString(w, fmt.Sprintf("git clean -fdx && "))
	io.WriteString(w, fmt.Sprintf("git reset --hard && "))
	io.WriteString(w, fmt.Sprintf("git remote set-url origin %s && ", b.RepoURL))
	io.WriteString(w, fmt.Sprintf("git fetch origin\n"))
	io.WriteString(w, fmt.Sprintf("else\n"))
	b.writeCloneCmd(w, builds_dir)
	io.WriteString(w, fmt.Sprintf("fi\n"))
}

func (b *Build) writeCheckoutCmd(w io.Writer, builds_dir string) {
	io.WriteString(w, fmt.Sprintf("echo Checkouting %s as %s...\n", b.Sha[0:8], b.RefName))
	io.WriteString(w, fmt.Sprintf("git checkout -B %s %s && ", b.RefName, b.Sha))
	io.WriteString(w, fmt.Sprintf("git reset --hard %s\n", b.Sha))
}

func (build *Build) Generate(builds_dir string, hostname string) ([]byte, error) {
	var b bytes.Buffer
	w := bufio.NewWriter(&b)

	io.WriteString(w, "#!/usr/bin/env bash\n")
	io.WriteString(w, "\n")
	if len(hostname) != 0 {
		io.WriteString(w, fmt.Sprintf("echo Running $(hostname) on %s...\n", ShellEscape(hostname)))
	} else {
		io.WriteString(w, "echo Running $(hostname)...\n")
	}
	io.WriteString(w, "\n")
	io.WriteString(w, "trap 'kill -s INT 0' EXIT\n")

	io.WriteString(w, "\n")
	if build.AllowGitFetch {
		build.writeFetchCmd(w, builds_dir)
	} else {
		build.writeCloneCmd(w, builds_dir)
	}

	build.writeCheckoutCmd(w, builds_dir)
	io.WriteString(w, "\n")
	io.WriteString(w, "set -evo pipefail\n")
	io.WriteString(w, "\n")

	commands := build.Commands
	commands = strings.Replace(commands, "\r\n", "\n", -1)

	io.WriteString(w, commands)

	w.Flush()

	return b.Bytes(), nil
}

func (build *Build) GetEnv() []string {
	return []string{
		fmt.Sprintf("CI_BUILD_REF=%s", build.Sha),
		fmt.Sprintf("CI_BUILD_BEFORE_SHA=%s", build.BeforeSha),
		fmt.Sprintf("CI_BUILD_REF_NAME=%s", build.RefName),
		fmt.Sprintf("CI_BUILD_ID=%d", build.Id),
		fmt.Sprintf("CI_BUILD_REPO=%s", build.RepoURL),
		fmt.Sprintf("CI_PROJECT_ID=%d", build.ProjectId),
		"CI_SERVER=yes",
		"CI_SERVER_NAME=GitLab CI",
		"CI_SERVER_VERSION=",
		"CI_SERVER_REVISION=",
		"RUBYLIB=",
		"RUBYOPT=",
		"BNDLE_BIN_PATH=",
		"BUNDLE_GEMFILE=",
	}
}

func (build *Build) fail(err error) {
	log.Errorln(build.Runner.ShortDescription(), build.Id, "Build failed", err)
	for {
		error_buffer := bytes.NewBufferString(err.Error())
		result := UpdateBuild(*build.Runner, build.Id, Failed, error_buffer)
		switch result {
		case UpdateSucceeded:
			return
		case UpdateAbort:
			return
		case UpdateFailed:
			time.Sleep(UPDATE_RETRY_INTERVAL * time.Second)
			continue
		}
	}
}

func (build *Build) Run() error {
	var err error
	executor := GetExecutor(build.Runner.Executor)
	if executor == nil {
		err = errors.New("executor not found")
	}
	if err == nil {
		err = executor.Prepare(build.Runner, build)
	}
	if err == nil {
		err = executor.Start()
	}
	if err == nil {
		err = executor.Wait()
	}
	if err != nil {
		go build.fail(err)
	}
	if executor != nil {
		executor.Cleanup()
	}
	return err
}
