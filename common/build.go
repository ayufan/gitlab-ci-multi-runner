package common

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"time"
)

type BuildState string

type GitStrategy int

const (
	GitClone GitStrategy = iota
	GitFetch
)

const (
	Pending BuildState = "pending"
	Running            = "running"
	Failed             = "failed"
	Success            = "success"
)

type Build struct {
	GetBuildResponse `yaml:",inline"`

	Trace           BuildTrace
	SystemInterrupt chan os.Signal `json:"-" yaml:"-"`
	RootDir         string         `json:"-" yaml:"-"`
	BuildDir        string         `json:"-" yaml:"-"`
	CacheDir        string         `json:"-" yaml:"-"`
	Hostname        string         `json:"-" yaml:"-"`
	Runner          *RunnerConfig  `json:"runner"`
	ExecutorData    ExecutorData

	// Unique ID for all running builds on this runner
	RunnerID int `json:"runner_id"`

	// Unique ID for all running builds on this runner and this project
	ProjectRunnerID int `json:"project_runner_id"`
}

func (b *Build) Log() *logrus.Entry {
	return b.Runner.Log().WithField("build", b.ID).WithField("project", b.ProjectID)
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

func (b *Build) executeShellScript(scriptType ShellScriptType, executor Executor, abort chan interface{}) error {
	shell := executor.Shell()
	if shell == nil {
		return errors.New("No shell defined")
	}

	script, err := GenerateShellScript(scriptType, *shell)
	if err != nil {
		return err
	}

	// Nothing to execute
	if script == "" {
		return nil
	}

	cmd := ExecutorCommand{
		Script: script,
		Abort:  abort,
	}

	switch scriptType {
	case ShellBuildScript, ShellAfterScript: // use custom build environment
		cmd.Predefined = false
	default: // all other stages use a predefined build environment
		cmd.Predefined = true
	}

	return executor.Run(cmd)
}

func (b *Build) executeUploadArtifacts(state error, executor Executor, abort chan interface{}) (err error) {
	when, _ := b.Options.GetString("artifacts", "when")

	if state == nil {
		// Previous stages were successful
		if when == "" || when == "on_success" || when == "always" {
			err = b.executeShellScript(ShellUploadArtifacts, executor, abort)
		}
	} else {
		// Previous stage did fail
		if when == "on_failure" || when == "always" {
			err = b.executeShellScript(ShellUploadArtifacts, executor, abort)
		}
	}

	// Use previous error if set
	if state != nil {
		err = state
	}
	return
}

func (b *Build) executeScript(executor Executor, abort chan interface{}) error {
	// Execute pre script (git clone, cache restore, artifacts download)
	err := b.executeShellScript(ShellPrepareScript, executor, abort)

	if err == nil {
		// Execute user build script (before_script + script)
		err = b.executeShellScript(ShellBuildScript, executor, abort)

		// Execute after script (after_script)
		timeoutCh := make(chan interface{}, 1)
		timeout := time.AfterFunc(time.Minute*5, func() {
			close(timeoutCh)
		})
		b.executeShellScript(ShellAfterScript, executor, timeoutCh)
		timeout.Stop()
	}

	// Execute post script (cache store, artifacts upload)
	if err == nil {
		err = b.executeShellScript(ShellArchiveCache, executor, abort)
	}
	err = b.executeUploadArtifacts(err, executor, abort)
	return err
}

func (b *Build) run(executor Executor) (err error) {
	buildTimeout := b.Timeout
	if buildTimeout <= 0 {
		buildTimeout = DefaultTimeout
	}

	buildFinish := make(chan error, 1)
	buildAbort := make(chan interface{})

	// Run build script
	go func() {
		buildFinish <- b.executeScript(executor, buildAbort)
	}()

	// Wait for signals: cancel, timeout, abort or finish
	b.Log().Debugln("Waiting for signals...")
	select {
	case <-b.Trace.Aborted():
		err = &BuildError{Inner: errors.New("canceled")}

	case <-time.After(time.Duration(buildTimeout) * time.Second):
		err = &BuildError{Inner: fmt.Errorf("execution took longer than %v seconds", buildTimeout)}

	case signal := <-b.SystemInterrupt:
		err = fmt.Errorf("aborted: %v", signal)

	case err = <-buildFinish:
		return err
	}

	b.Log().WithError(err).Debugln("Waiting for build to finish...")

	// Wait till we receive that build did finish
	for {
		select {
		case buildAbort <- true:
		case <-buildFinish:
			return err
		}
	}
}

func (b *Build) retryCreateExecutor(globalConfig *Config, provider ExecutorProvider, logger BuildLogger) (executor Executor, err error) {
	for tries := 0; tries < PreparationRetries; tries++ {
		executor = provider.Create()
		if executor == nil {
			err = errors.New("failed to create executor")
			return
		}

		err = executor.Prepare(globalConfig, b.Runner, b)
		if err == nil {
			break
		}
		if executor != nil {
			executor.Cleanup()
			executor = nil
		}

		logger.SoftErrorln("Preparation failed:", err)
		logger.Infoln("Will be retried in", PreparationRetryInterval, "...")
		time.Sleep(PreparationRetryInterval)
	}
	return
}

func (b *Build) Run(globalConfig *Config, trace BuildTrace) (err error) {
	var executor Executor

	logger := NewBuildLogger(trace, b.Log())
	logger.Println("Running with " + AppVersion.Line() + helpers.ANSI_RESET)

	defer func() {
		if _, ok := err.(*BuildError); ok {
			logger.SoftErrorln("Build failed:", err)
			trace.Fail(err)
		} else if err != nil {
			logger.Errorln("Build failed (system failure):", err)
			trace.Fail(err)
		} else {
			logger.Infoln("Build succeeded")
			trace.Success()
		}
		if executor != nil {
			executor.Cleanup()
		}
	}()

	b.Trace = trace

	provider := GetExecutor(b.Runner.Executor)
	if provider == nil {
		return errors.New("executor not found")
	}

	executor, err = b.retryCreateExecutor(globalConfig, provider, logger)
	if err == nil {
		err = b.run(executor)
	}
	if executor != nil {
		executor.Finish(err)
	}
	return err
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
		{"CI_BUILD_TOKEN", b.Token, true, true, false},
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

func (b *Build) GetGitDepth() string {
	return b.GetAllVariables().Get("GIT_DEPTH")
}

func (b *Build) GetGitStrategy() GitStrategy {
	switch b.GetAllVariables().Get("GIT_STRATEGY") {
	case "clone":
		return GitClone

	case "fetch":
		return GitFetch

	default:
		if b.AllowGitFetch {
			return GitFetch
		}

		return GitClone
	}
}
