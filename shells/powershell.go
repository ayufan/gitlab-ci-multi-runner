package shells

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/ayufan/gitlab-ci-multi-runner/common"
	"github.com/ayufan/gitlab-ci-multi-runner/helpers"
	"io"
	"path/filepath"
	"strings"
)

type PowerShell struct {
}

func (c *PowerShell) GetName() string {
	return "powershell"
}

func (b *PowerShell) writeCommand(w io.Writer, format string, args ...interface{}) {
	io.WriteString(w, fmt.Sprintf(format, args...)+"\r\n")
}

func (b *PowerShell) writeCommandChecked(w io.Writer, format string, args ...interface{}) {
	b.writeCommand(w, format, args...)
}

func (b *PowerShell) writeCloneCmd(w io.Writer, build *common.Build) {
	b.writeCommand(w, "echo \"Clonning repository...\"")
	b.writeCommandChecked(w, "(Test-Path \"%s\") -or (New-Item \"%s\")", filepath.FromSlash(build.BuildsDir), filepath.FromSlash(build.BuildsDir))
	b.writeCommandChecked(w, "cd %s", filepath.FromSlash(build.BuildsDir))
	b.writeCommandChecked(w, "if(Test-Path \"%s\") { Remove-Item -Recurse %s }", filepath.FromSlash(build.ProjectDir()))
	b.writeCommandChecked(w, "git clone %s %s", build.RepoURL, filepath.FromSlash(build.ProjectDir()))
	b.writeCommandChecked(w, "cd %s", filepath.FromSlash(build.ProjectDir()))
}

func (b *PowerShell) writeFetchCmd(w io.Writer, build *common.Build) {
	b.writeCommand(w, "if(Test-Path \"%s\\%s\\.git\") {", filepath.FromSlash(build.BuildsDir), filepath.FromSlash(build.ProjectDir()))
	b.writeCommand(w, "echo \"Fetching changes...\"")
	b.writeCommandChecked(w, "cd %s\\%s", filepath.FromSlash(build.BuildsDir), filepath.FromSlash(build.ProjectDir()))
	b.writeCommandChecked(w, "git clean -fdx")
	b.writeCommandChecked(w, "git reset --hard > $null")
	b.writeCommandChecked(w, "git remote set-url origin %s", build.RepoURL)
	b.writeCommandChecked(w, "git fetch origin")
	b.writeCommand(w, "} else {")
	b.writeCloneCmd(w, build)
	b.writeCommand(w, "}")
}

func (b *PowerShell) writeCheckoutCmd(w io.Writer, build *common.Build) {
	b.writeCommand(w, "echo \"Checkouting %s as %s...\"", build.Sha[0:8], build.RefName)
	b.writeCommandChecked(w, "git checkout -B %s %s > $null", build.RefName, build.Sha)
	b.writeCommandChecked(w, "git reset --hard %s > $null", build.Sha)
}

func (b *PowerShell) GenerateScript(build *common.Build) (*common.ShellScript, error) {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)

	b.writeCommand(w, "$ErrorActionPreference = \"Stop\"")

	if len(build.Hostname) != 0 {
		b.writeCommand(w, "echo \"Running on $env:computername via %s...\"", helpers.ShellEscape(build.Hostname))
	} else {
		b.writeCommand(w, "echo \"Running on $env:computername...\"")
	}
	b.writeCommand(w, "")

	if build.AllowGitFetch {
		b.writeFetchCmd(w, build)
	} else {
		b.writeCloneCmd(w, build)
	}

	b.writeCheckoutCmd(w, build)
	b.writeCommand(w, "")

	for _, command := range strings.Split(build.Commands, "\n") {
		command = strings.TrimRight(command, " \t\r\n")
		if strings.TrimSpace(command) == "" {
			b.writeCommand(w, "echo \"\"")
			continue
		}

		if !build.Runner.DisableVerbose {
			b.writeCommand(w, "echo \"%s\"", command)
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

		"CI_SERVER=yes",
		"CI_SERVER_NAME=GitLab CI",
		"CI_SERVER_VERSION=",
		"CI_SERVER_REVISION=",
	}

	script := common.ShellScript{
		Environment: env,
		Script:      buffer.Bytes(),
		Command:     "powershell",
		Arguments:   []string{"-noprofile", "-noninteractive", "-executionpolicy", "Bypass", "-command"},
		PassFile:    true,
		Extension:   "ps1",
	}
	return &script, nil
}

func (b *PowerShell) IsDefault() bool {
	return false
}

func init() {
	common.RegisterShell(&PowerShell{})
}
