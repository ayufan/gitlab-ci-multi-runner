package shells

import (
	"bufio"
	"bytes"
	"fmt"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"io"
	"path/filepath"
	"runtime"
	"strings"
)

type CmdShell struct {
}

func (b *CmdShell) GetName() string {
	return "cmd"
}

func (b *CmdShell) writeCommand(w io.Writer, format string, args ...interface{}) {
	io.WriteString(w, fmt.Sprintf(format, args...)+"\r\n")
}

func (b *CmdShell) writeCommandChecked(w io.Writer, format string, args ...interface{}) {
	b.writeCommand(w, format, args...)
	b.writeCommand(w, "%s", "IF %errorlevel% NEQ 0 exit /b %errorlevel%")
}

func (b *CmdShell) writeCloneCmd(w io.Writer, build *common.Build) {
	dir := filepath.FromSlash(build.FullProjectDir())
	b.writeCommand(w, "echo Clonning repository...")
	b.writeCommandChecked(w, "rd /s /q \"%s\" 2> NUL 1>NUL", dir)
	b.writeCommandChecked(w, "md \"%s\"", dir)
	b.writeCommandChecked(w, "git clone \"%s\" \"%s\"", build.RepoURL, dir)
	b.writeCommandChecked(w, "cd \"%s\"", dir)
}

func (b *CmdShell) writeFetchCmd(w io.Writer, build *common.Build) {
	dir := filepath.FromSlash(build.FullProjectDir())
	b.writeCommand(w, "IF EXIST \"%s\\.git\" (", dir)
	b.writeCommand(w, "echo Fetching changes...")
	b.writeCommandChecked(w, "cd \"%s\"", dir)
	b.writeCommandChecked(w, "git clean -fdx")
	b.writeCommandChecked(w, "git reset --hard > NUL")
	b.writeCommandChecked(w, "git remote set-url origin \"%s\"", build.RepoURL)
	b.writeCommandChecked(w, "git fetch origin")
	b.writeCommand(w, ") ELSE (")
	b.writeCloneCmd(w, build)
	b.writeCommand(w, ")")
}

func (b *CmdShell) writeCheckoutCmd(w io.Writer, build *common.Build) {
	b.writeCommand(w, "echo Checkouting %s as %s...", build.Sha[0:8], build.RefName)
	b.writeCommandChecked(w, "git checkout -qf \"%s\"", build.Sha)
}

func (b *CmdShell) GenerateScript(build *common.Build, shellType common.ShellType) (*common.ShellScript, error) {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)

	b.writeCommand(w, "@echo off")
	b.writeCommand(w, "echo.")
	b.writeCommand(w, "setlocal enableextensions")

	if len(build.Hostname) != 0 {
		b.writeCommand(w, "echo Running on %s via %s...", "%COMPUTERNAME%", helpers.ShellEscape(build.Hostname))
	} else {
		b.writeCommand(w, "echo Running on %s...", "%COMPUTERNAME%")
	}

	if build.AllowGitFetch {
		b.writeFetchCmd(w, build)
	} else {
		b.writeCloneCmd(w, build)
	}

	b.writeCheckoutCmd(w, build)

	for _, command := range strings.Split(build.Commands, "\n") {
		command = strings.TrimRight(command, " \t\r\n")
		if strings.TrimSpace(command) == "" {
			b.writeCommand(w, "echo.")
			continue
		}

		if !helpers.BoolOrDefault(build.Runner.DisableVerbose, false) {
			b.writeCommand(w, "echo %s", command)
		}
		b.writeCommandChecked(w, "%s", command)
	}

	w.Flush()

	env := []string{
		fmt.Sprintf("CI_BUILD_REF=%s", build.Sha),
		fmt.Sprintf("CI_BUILD_BEFORE_SHA=%s", build.BeforeSha),
		fmt.Sprintf("CI_BUILD_REF_NAME=%s", build.RefName),
		fmt.Sprintf("CI_BUILD_ID=%d", build.ID),
		fmt.Sprintf("CI_BUILD_REPO=%s", build.RepoURL),

		fmt.Sprintf("CI_PROJECT_ID=%d", build.ProjectID),
		fmt.Sprintf("CI_PROJECT_DIR=%s", filepath.FromSlash(build.FullProjectDir())),

		"CI=true",
		"CI_SERVER=yes",
		"CI_SERVER_NAME=GitLab CI",
		"CI_SERVER_VERSION=",
		"CI_SERVER_REVISION=",

		"GITLAB_CI=true",
	}

	script := common.ShellScript{
		Environment: env,
		Script:      buffer.Bytes(),
		Command:     "cmd",
		Arguments:   []string{"/Q", "/C"},
		PassFile:    true,
		Extension:   "cmd",
	}
	return &script, nil
}

func (b *CmdShell) IsDefault() bool {
	return runtime.GOOS == "windows"
}

func init() {
	common.RegisterShell(&CmdShell{})
}
