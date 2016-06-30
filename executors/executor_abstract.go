package executors

import (
	"os"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

type ExecutorOptions struct {
	DefaultBuildsDir string
	DefaultCacheDir  string
	SharedBuildsDir  bool
	Shell            common.ShellScriptInfo
	ShowHostname     bool
	SupportedOptions []string
}

type AbstractExecutor struct {
	ExecutorOptions
	common.BuildLogger
	Config     common.RunnerConfig
	Build      *common.Build
	BuildTrace common.BuildTrace
	BuildShell *common.ShellConfiguration
}

func (e *AbstractExecutor) updateShell() error {
	script := e.Shell()
	script.Build = e.Build
	if e.Config.Shell != "" {
		script.Shell = e.Config.Shell
	}
	return nil
}

func (e *AbstractExecutor) generateShellConfiguration() error {
	shellConfiguration, err := common.GetShellConfiguration(*e.Shell())
	if err != nil {
		return err
	}
	e.BuildShell = shellConfiguration
	e.Debugln("Shell configuration:", shellConfiguration)
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

func (e *AbstractExecutor) verifyOptions() error {
	supportedOptions := e.SupportedOptions
	if shell := common.GetShell(e.Shell().Shell); shell != nil {
		supportedOptions = append(supportedOptions, shell.GetSupportedOptions()...)
	}

	for key, value := range e.Build.Options {
		if value == nil {
			continue
		}
		found := false
		for _, option := range supportedOptions {
			if option == key {
				found = true
				break
			}
		}

		if !found {
			e.Warningln(key, "is not supported by selected executor and shell")
		}
	}
	return nil
}

func (e *AbstractExecutor) Shell() *common.ShellScriptInfo {
	return &e.ExecutorOptions.Shell
}

func (e *AbstractExecutor) Prepare(globalConfig *common.Config, config *common.RunnerConfig, build *common.Build) error {
	e.Config = *config
	e.Build = build
	e.BuildTrace = build.Trace
	e.BuildLogger = common.NewBuildLogger(build.Trace, build.Log())

	err := e.startBuild()
	if err != nil {
		return err
	}

	err = e.updateShell()
	if err != nil {
		return err
	}

	err = e.verifyOptions()
	if err != nil {
		return err
	}

	err = e.generateShellConfiguration()
	if err != nil {
		return err
	}
	return nil
}

func (e *AbstractExecutor) Finish(err error) {
}

func (e *AbstractExecutor) Cleanup() {
}
