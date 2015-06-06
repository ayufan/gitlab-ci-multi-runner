package shells

import (
	"bufio"
	"bytes"
	"fmt"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"io"
	"strings"
)

type PowerShell struct {
	AbstractShell
}

func (b *PowerShell) GetName() string {
	return "powershell"
}

func (b *PowerShell) writeCommand(w io.Writer, format string, args ...interface{}) {
	io.WriteString(w, fmt.Sprintf(format, args...)+"\r\n")
}

func (b *PowerShell) writeCommandChecked(w io.Writer, format string, args ...interface{}) {
	b.writeCommand(w, format, args...)
}

func (b *PowerShell) writeCloneCmd(w io.Writer, build *common.Build, dir string) {
	b.writeCommand(w, "echo \"Cloning repository...\"")
	b.writeCommandChecked(w, "if(Test-Path \"%s\") { Remove-Item -Force -Recurse \"%s\" }", dir, dir)
	b.writeCommandChecked(w, "(Test-Path \"%s\") -or (New-Item \"%s\")", dir, dir)
	b.writeCommandChecked(w, "git clone \"%s\" \"%s\"", build.RepoURL, dir)
	b.writeCommandChecked(w, "cd \"%s\"", dir)
}

func (b *PowerShell) writeFetchCmd(w io.Writer, build *common.Build, dir string) {
	b.writeCommand(w, "if(Test-Path \"%s\\.git\") {", dir)
	b.writeCommand(w, "echo \"Fetching changes...\"")
	b.writeCommandChecked(w, "cd \"%s\"", dir)
	b.writeCommandChecked(w, "git clean -fdx")
	b.writeCommandChecked(w, "git reset --hard > $null")
	b.writeCommandChecked(w, "git remote set-url origin \"%s\"", build.RepoURL)
	b.writeCommandChecked(w, "git fetch origin")
	b.writeCommand(w, "} else {")
	b.writeCloneCmd(w, build, dir)
	b.writeCommand(w, "}")
}

func (b *PowerShell) writeCheckoutCmd(w io.Writer, build *common.Build) {
	b.writeCommand(w, "echo \"Checking out %s as %s...\"", build.Sha[0:8], build.RefName)
	b.writeCommandChecked(w, "git checkout -qf \"%s\"", build.Sha)
}

func (b *PowerShell) GenerateScript(build *common.Build, shellType common.ShellType) (*common.ShellScript, error) {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)

	projectDir := build.FullProjectDir()
	projectDir = helpers.ToBackslash(projectDir)

	b.writeCommand(w, "$ErrorActionPreference = \"Stop\"")

	if len(build.Hostname) != 0 {
		b.writeCommand(w, "echo \"Running on $env:computername via %s...\"", helpers.ShellEscape(build.Hostname))
	} else {
		b.writeCommand(w, "echo \"Running on $env:computername...\"")
	}
	b.writeCommand(w, "")

	if build.AllowGitFetch {
		b.writeFetchCmd(w, build, projectDir)
	} else {
		b.writeCloneCmd(w, build, projectDir)
	}

	b.writeCheckoutCmd(w, build)
	b.writeCommand(w, "")

	for _, command := range strings.Split(build.Commands, "\n") {
		command = strings.TrimRight(command, " \t\r\n")
		if strings.TrimSpace(command) == "" {
			b.writeCommand(w, "echo \"\"")
			continue
		}

		if !helpers.BoolOrDefault(build.Runner.DisableVerbose, false) {
			b.writeCommand(w, "echo \"%s\"", command)
		}
		b.writeCommandChecked(w, "%s", command)
	}

	w.Flush()

	script := common.ShellScript{
		Environment: b.GetVariables(build, projectDir),
		Script:      buffer.String(),
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
