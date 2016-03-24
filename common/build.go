package common

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
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
	GetBuildResponse `yaml:",inline"`

	Runner   RunnerConfig
	Trace    BuildTrace
	Executor Executor
	Logging

	RootDir  string
	BuildDir string
	CacheDir string
	Hostname string

	Shell         Shell
	ShellType     ShellType
	RunnerCommand string

	// Unique ID for all running builds on this runner
	RunnerID int

	// Unique ID for all running builds on this runner and this project
	ProjectRunnerID int
}

func (b *Build) ProjectUniqueName() string {
	return fmt.Sprintf("runner-%s-project-%d-concurrent-%d",
		b.Runner.ShortDescription(), b.ProjectID, b.ProjectRunnerID)
}

func (b *Build) ProjectSlug() (string, error) {
	url, err := url.Parse(b.RepoURL)
	if err != nil {
		return "", err
	}
	if url.Host == "" {
		return "", errors.New("only URI reference supported")
	}

	slug := url.Path
	slug = strings.TrimSuffix(slug, ".git")
	slug = path.Clean(slug)
	if slug == "." {
		return "", errors.New("invalid path")
	}
	if strings.Contains(slug, "..") {
		return "", errors.New("it doesn't look like a valid path")
	}
	return slug, nil
}

func (b *Build) ProjectUniqueDir(sharedDir bool) string {
	dir, err := b.ProjectSlug()
	if err != nil {
		dir = fmt.Sprintf("project-%d", b.ProjectID)
	}

	// for shared dirs path is constructed like this:
	// <some-path>/runner-short-id/concurrent-id/group-name/project-name/
	// ex.<some-path>/01234567/0/group/repo/
	if sharedDir {
		dir = path.Join(
			fmt.Sprintf("%s", b.Runner.ShortDescription()),
			fmt.Sprintf("%d", b.ProjectRunnerID),
			dir,
		)
	}
	return dir
}

func (b *Build) FullProjectDir() string {
	return helpers.ToSlash(b.BuildDir)
}

func (b *Build) StartBuild(rootDir, cacheDir string, sharedDir bool) {
	b.RootDir = rootDir
	b.BuildDir = path.Join(rootDir, b.ProjectUniqueDir(sharedDir))
	b.CacheDir = path.Join(cacheDir, b.ProjectUniqueDir(false))
}

func (b *Build) wait(trace BuildTrace, done chan struct{}, signal chan os.Signal, abort chan error) {
	defer close(abort)

	cancel := make(chan struct{}, 1)
	trace.Notify(func() {
		cancel <- struct{}{}
	})

	buildTimeout := b.Timeout
	if buildTimeout <= 0 {
		buildTimeout = DefaultTimeout
	}

	b.Debugln("Waiting for signals...")
	select {
	case <-time.After(time.Duration(buildTimeout) * time.Second):
		abort <- fmt.Errorf("execution took longer than %v seconds", buildTimeout)

	case signal := <-signal:
		abort <- fmt.Errorf("aborted: %v", signal)

	case <-cancel:
		abort <- fmt.Errorf("canceled")

	case <-done:
		return
	}
}

func (b *Build) Run(data ExecutorData, trace BuildTrace, signal chan os.Signal) (err error) {
	if b.Executor != nil {
		return errors.New("object already used")
	}

	b.Trace = trace
	b.Logging = Logging{
		LogEntry: b.Runner.Log().WithField("build", b.ID),
		LogTrace: trace,
	}

	defer func() {
		if err != nil {
			b.Println()
			b.Errorln("Build failed:", err)
			trace.Fail(err)
		} else {
			b.Println()
			b.Infoln("Build succeeded")
			trace.Success()
		}
	}()

	b.Executor = NewExecutor(b.Runner.Executor)
	if b.Executor == nil {
		return errors.New("executor not found")
	}
	defer b.Executor.Cleanup()

	err = b.Executor.Prepare(b, data)
	if err != nil {
		return
	}

	plugin := GetPlugin("default")
	if plugin == nil {
		return errors.New("plugin not found")
	}

	done := make(chan struct{})
	defer close(done)

	abort := make(chan error)

	// Wait for signals: cancel, timeout, abort or finish
	go b.wait(trace, done, signal, abort)

	return plugin.Run(b, abort)
}

func (b *Build) Step(script *ShellScript, image string, abort chan error) error {
	if b.Executor == nil {
		return errors.New("build not started")
	}

	run := ExecutorRun{
		ShellScript: *script,
		Image:       image,
		Abort:       abort,
		Trace:       b.Trace,
	}
	return b.Executor.Run(run)
}

func (b *Build) String() string {
	return helpers.ToYAML(b)
}

func (b *Build) GetDefaultVariables() BuildVariables {
	return BuildVariables{
		{"CI", "true", true, true, false},
		{"CI_BUILD_REF", b.Sha, true, true, false},
		{"CI_BUILD_BEFORE_SHA", b.BeforeSha, true, true, false},
		{"CI_BUILD_REF_NAME", b.RefName, true, true, false},
		{"CI_BUILD_ID", strconv.Itoa(b.ID), true, true, false},
		{"CI_BUILD_REPO", b.RepoURL, true, true, false},
		{"CI_PROJECT_ID", strconv.Itoa(b.ProjectID), true, true, false},
		{"CI_PROJECT_DIR", b.FullProjectDir(), true, true, false},
		{"CI_SERVER", "yes", true, true, false},
		{"CI_SERVER_NAME", "GitLab CI", true, true, false},
		{"CI_SERVER_VERSION", "", true, true, false},
		{"CI_SERVER_REVISION", "", true, true, false},
		{"GITLAB_CI", "true", true, true, false},
	}
}

func (b *Build) GetAllVariables() BuildVariables {
	variables := b.Runner.GetVariables()
	variables = append(variables, b.GetDefaultVariables()...)
	variables = append(variables, b.Variables...)
	return variables.Expand()
}

func NewBuild(response GetBuildResponse, runner RunnerConfig) *Build {
	return &Build{
		GetBuildResponse: response,
		Runner:           runner,
	}
}
