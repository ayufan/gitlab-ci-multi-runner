package src

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

type Build struct {
	GetBuildResponse
	Name     string
	BuildLog string
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
	io.WriteString(w, fmt.Sprintf("mkdir -p %s && ", builds_dir))
	io.WriteString(w, fmt.Sprintf("cd %s && ", builds_dir))
	io.WriteString(w, fmt.Sprintf("rm -rf %s && ", b.ProjectDir()))
	io.WriteString(w, fmt.Sprintf("git clone %s %s && ", b.RepoURL, b.ProjectDir()))
	io.WriteString(w, fmt.Sprintf("cd %s\n", b.ProjectDir()))
}

func (b *Build) writeFetchCmd(w io.Writer, builds_dir string) {
	io.WriteString(w, fmt.Sprintf("if [[ -d %s/%s/.git ]]; then\n", builds_dir, b.ProjectDir()))
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
	io.WriteString(w, fmt.Sprintf("git checkout %s && ", b.RefName))
	io.WriteString(w, fmt.Sprintf("git reset --hard %s\n", b.Sha))
}

func (build *Build) Generate(builds_dir string) ([]byte, error) {
	var b bytes.Buffer
	w := bufio.NewWriter(&b)

	io.WriteString(w, "#!/usr/bin/env bash\n")
	io.WriteString(w, "\n")
	io.WriteString(w, "echo Using $(hostname)\n")
	io.WriteString(w, "\n")
	io.WriteString(w, "trap 'kill -s INT 0' EXIT\n")
	io.WriteString(w, "set -evo pipefail\n")
	io.WriteString(w, "\n")

	if build.AllowGitFetch {
		build.writeFetchCmd(w, builds_dir)
	} else {
		build.writeCloneCmd(w, builds_dir)
	}

	build.writeCheckoutCmd(w, builds_dir)
	io.WriteString(w, "\n")

	commands := build.Commands
	commands = strings.Replace(commands, "\r\n", "\n", -1)

	io.WriteString(w, commands)

	w.Flush()

	return b.Bytes(), nil
}

func (b *Build) ReadBuildLog() io.ReadCloser {
	file, err := os.Open(b.BuildLog)
	if err != nil {
		return nil
	}
	return file
}

func (b *Build) WatchTrace(config RunnerConfig, abort chan bool, finished chan bool) {
	for {
		select {
		case <-time.After(UPDATE_INTERVAL * time.Second):
			file := b.ReadBuildLog()
			if file == nil {
				continue
			}
			defer file.Close()
			switch UpdateBuild(config, b.Id, Running, file) {
			case UpdateSucceeded:
			case UpdateAbort:
				log.Debugln(config.ShortDescription(), b.Id, "updateBuildLog", "Sending abort request...")
				abort <- true
				log.Debugln(config.ShortDescription(), b.Id, "updateBuildLog", "Waiting for finished flag...")
				<-finished
				log.Debugln(config.ShortDescription(), b.Id, "updateBuildLog", "Thread finished.")
				return
			case UpdateFailed:
			}

		case <-finished:
			log.Debugln(config.ShortDescription(), b.Id, "updateBuildLog", "Received finish.")
			return
		}
	}
}

func (b *Build) FinishBuild(config RunnerConfig, buildState BuildState, extraMessage string) {
	build, _ := ioutil.ReadFile(b.BuildLog)

	go func() {
		for {
			buffer := io.MultiReader(bytes.NewReader(build), bytes.NewBufferString(extraMessage))
			if UpdateBuild(config, b.Id, buildState, buffer) != UpdateFailed {
				break
			} else {
				time.Sleep(UPDATE_RETRY_INTERVAL * time.Second)
			}
		}

		log.Println(config.ShortDescription(), b.Id, "Build finished.")
	}()
}

func (build *Build) GetEnv() []string {
	return []string{
		"CI_SERVER=yes",
		"CI_SERVER_NAME=GitLab CI",
		"CI_SERVER_VERSION=",
		"CI_SERVER_REVISION=",
		fmt.Sprintf("CI_BUILD_REF=%s", build.Sha),
		fmt.Sprintf("CI_BUILD_BEFORE_SHA=%s", build.BeforeSha),
		fmt.Sprintf("CI_BUILD_REF_NAME=%s", build.RefName),
		fmt.Sprintf("CI_BUILD_ID=%d", build.Id),
		fmt.Sprintf("CI_BUILD_REPO=%s", build.RepoURL),
		fmt.Sprintf("CI_PROJECT_ID=%d", build.ProjectId),
		"RUBYLIB=",
		"RUBYOPT=",
		"BNDLE_BIN_PATH=",
		"BUNDLE_GEMFILE=",
	}
}

func (build *Build) CreateBuildLog() (io.WriteCloser, error) {
	// create build log
	build_log, err := ioutil.TempFile("", "build_log")
	if err != nil {
		return nil, errors.New("Failed to create build log file")
	}
	build.BuildLog = build_log.Name()
	log.Debugln(build.Id, "Created build log:", build_log.Name())
	return build_log, nil
}

func (build *Build) DeleteBuildLog() {
	if len(build.BuildLog) != 0 {
		os.Remove(build.BuildLog)
		build.BuildLog = ""
	}
}
