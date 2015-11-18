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
	b.writeCommand(w, "%s", "if (!$?) { Exit $LASTEXITCODE }")
}

func (b *PowerShell) writeCloneCmd(w io.Writer, build *common.Build, dir string) {
	b.writeCommand(w, "echo \"Cloning repository...\"")
	b.writeCommand(w, "Import-Module -Name NTFSSecurity -ErrorAction SilentlyContinue")
	b.writeCommand(w, "if( (Get-Command -Name Remove-Item2 -Module NTFSSecurity -ErrorAction SilentlyContinue) -and (Test-Path \"%s\") ) {", dir)
	b.writeCommandChecked(w, "Remove-Item2 -Force -Recurse \"%s\"", dir)
	b.writeCommand(w, "} elseif(Test-Path \"%s\") {", dir)
	b.writeCommandChecked(w, "Remove-Item -Force -Recurse \"%s\"", dir)
	b.writeCommand(w, "}")
	b.writeCommandChecked(w, "(Test-Path \"%s\") -or (New-Item \"%s\" -ItemType \"directory\" )", dir, dir)
	b.writeCommandChecked(w, "git clone \"%s\" \"%s\"", build.RepoURL, dir)
	b.writeCommandChecked(w, "cd \"%s\"", dir)
}

func (b *PowerShell) writeFetchCmd(w io.Writer, build *common.Build, dir string) {
	b.writeCommand(w, "if(Test-Path \"%s\\.git\") {", dir)
	b.writeCommand(w, "echo \"Fetching changes...\"")
	b.writeCommandChecked(w, "cd \"%s\"", dir)
	b.writeCommandChecked(w, "git clean -ffdx")
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

func (b *PowerShell) GenerateScript(info common.ShellScriptInfo) (*common.ShellScript, error) {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)

	build := info.Build
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
			b.writeCommand(w, "echo \"%s\"", strings.Replace(command, "\"", "`\"", -1))
		}
		b.writeCommandChecked(w, "%s", command)
	}

	w.Flush()

	script := common.ShellScript{
		Environment: b.GetVariables(build, projectDir, info.Environment),
		BuildScript: buffer.String(),
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
