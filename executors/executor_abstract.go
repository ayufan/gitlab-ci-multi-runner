package executors

import (
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"os"
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
	Config      common.RunnerConfig
	Build       *common.Build
	BuildLog    common.BuildTrace
	BuildScript *common.ShellScript
}

func (e *AbstractExecutor) updateShell() error {
	script := &e.Shell
	script.Build = e.Build
	if e.Config.Shell != "" {
		script.Shell = e.Config.Shell
	}
	return nil
}

func (e *AbstractExecutor) generateShellScript() error {
	shellScript, err := common.GenerateShellScript(e.Shell)
	if err != nil {
		return err
	}
	e.BuildScript = shellScript
	e.Debugln("Shell script:", shellScript)
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
	if shell := common.GetShell(e.Shell.Shell); shell != nil {
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

func (e *AbstractExecutor) Prepare(globalConfig *common.Config, config *common.RunnerConfig, build *common.Build) error {
	e.Config = *config
	e.Build = build
	e.BuildLog = build.Trace

	err := e.startBuild()
	if err != nil {
		return err
	}

	e.Infoln(common.VersionLine())

	err = e.updateShell()
	if err != nil {
		return err
	}

	err = e.verifyOptions()
	if err != nil {
		return err
	}

	err = e.generateShellScript()
	if err != nil {
		return err
	}
	return nil
}

func (e *AbstractExecutor) ShellScript() *common.ShellScript {
	return e.BuildScript
}

func (e *AbstractExecutor) Finish(err error) {
	if err != nil {
		e.Println()
		e.Errorln("Build failed:", err)
	} else {
		e.Println()
		e.Infoln("Build succeeded")
	}
}

func (e *AbstractExecutor) Cleanup() {
}
