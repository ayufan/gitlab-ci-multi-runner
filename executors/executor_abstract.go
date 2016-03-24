package executors

import (
	"fmt"
	"os"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

type ExecutorOptions struct {
	DefaultBuildsDir string
	DefaultCacheDir  string
	SharedBuildsDir  bool
	Shell            string
	Type             common.ShellType
	RunnerCommand    string
	ShowHostname     bool
	SupportedOptions []string
}

type AbstractExecutor struct {
	ExecutorOptions

	common.Logging
	Config common.RunnerConfig
	Build  *common.Build
}

func (e *AbstractExecutor) updateShell() error {
	shell := e.Config.Shell
	if e.Config.Shell == "" {
		shell = e.Shell
	}

	e.Build.Shell = common.GetShell(shell)
	if e.Build.Shell == nil {
		return fmt.Errorf("shell not found: %v", shell)
	}

	e.Build.ShellType = e.Type
	e.Build.RunnerCommand = e.RunnerCommand
	return nil
}

func (e *AbstractExecutor) startBuild() error {
	// Save hostname
	if e.ShowHostname && e.Build.Hostname == "" {
		e.Build.Hostname, _ = os.Hostname()
	}

	// Start actual build
	rootDir := e.Config.BuildsDir
	if rootDir == "" {
		rootDir = e.DefaultBuildsDir
	}
	cacheDir := e.Config.CacheDir
	if cacheDir == "" {
		cacheDir = e.DefaultCacheDir
	}
	e.Build.StartBuild(rootDir, cacheDir, e.SharedBuildsDir)
	return nil
}

func (e *AbstractExecutor) Prepare(build *common.Build, data common.ExecutorData) error {
	e.Logging = build.Logging
	e.Config = build.Runner
	e.Build = build

	err := e.updateShell()
	if err != nil {
		return err
	}

	err = e.startBuild()
	if err != nil {
		return err
	}

	e.Infoln(fmt.Sprintf("%s %s (%s)", common.NAME, common.VERSION, common.REVISION))
	return nil
}

func (e *AbstractExecutor) Cleanup() {
}
