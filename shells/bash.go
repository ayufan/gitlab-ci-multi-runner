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
	"path"
	"strconv"
)

type BashShell struct {
	AbstractShell
}

func (b *BashShell) GetName() string {
	return "bash"
}

func (b *BashShell) executeCommand(w io.Writer, cmd string, arguments ...interface{}) {
	list := []string{
		helpers.ShellEscape(cmd),
	}

	for _, argument := range arguments {
		list = append(list, strconv.Quote(argument))
	}

	io.WriteString(w, strings.Join(list, " ") + "\n")
}

func (b *BashShell) executeCommandFormat(w io.Writer, format string, arguments ...string) {
	io.WriteString(w, fmt.Sprintf(format + "\n", arguments...))
}

func (b *BashShell) echoColored(w io.Writer, text string) {
	coloredText := helpers.ANSI_BOLD_GREEN + text + helpers.ANSI_RESET
	b.executeCommand(w, "echo", coloredText)
}

func (b *BashShell) echoWarning(w io.Writer, text string) {
	coloredText := helpers.ANSI_BOLD_YELLOW + text + helpers.ANSI_RESET
	b.executeCommand(w, "echo", coloredText)
}

func (b *BashShell) echoColoredFormat(w io.Writer, format string, a ...interface{}) {
	b.echoColored(w, fmt.Sprintf(format, a...))
}

func (b *BashShell) writeIfDirectory(w io.Writer, directory string) {
	b.executeCommandFormat(w, "if [[ -d %q ]]; then", directory)
}

func (b *BashShell) writeIfFile(w io.Writer, directory string) {
	b.executeCommandFormat(w, "if [[ -e %q ]]; then", directory)
}

func (b *BashShell) writeElse(w io.Writer) {
	b.executeCommandFormat(w, "else")
}

func (b *BashShell) writeEndIf(w io.Writer) {
	b.executeCommandFormat(w, "fi")
}

func (b *BashShell) writeCloneCmd(w io.Writer, build *common.Build, projectDir string) {
	b.echoColoredFormat(w, "Cloning repository...")
	b.executeCommand(w, "rm", "-rf", projectDir)
	b.executeCommand(w, "mkdir", "-p", projectDir)
	b.executeCommand(w, "git", "clone", build.RepoURL, projectDir)
	b.executeCommand(w, "cd", projectDir)
}

func (b *BashShell) writeFetchCmd(w io.Writer, build *common.Build, projectDir string, gitDir string) {
	b.writeIfDirectory(w, gitDir)
	b.echoColoredFormat(w, "Fetching changes...")
	b.executeCommand(w, "cd", projectDir)
	b.executeCommand(w, "git", "clean", "-ffdx")
	b.executeCommand(w, "git", "reset", "--hard")
	b.executeCommand(w, "git", "remote", "set-url", "origin", build.RepoURL)
	b.executeCommand(w, "git", "fetch", "origin")
	b.writeElse(w)
	b.writeCloneCmd(w, build, projectDir)
	b.writeEndIf(w)
}

func (b *BashShell) writeCheckoutCmd(w io.Writer, build *common.Build) {
	b.echoColoredFormat(w, "Checking out %s as %s...", build.Sha[0:8], build.RefName)
	b.executeCommand(w, "git", "checkout", build.Sha))
}

func (b *BashShell) writeCdBuildDir(w io.Writer, info common.ShellScriptInfo) {
	b.executeCommand(w, "cd", b.fullProjectDir(info))
}

func (b *BashShell) fullProjectDir(info common.ShellScriptInfo) string {
	projectDir := info.Build.FullProjectDir()
	return helpers.ToSlash(projectDir)
}

func (b *BashShell) generateExports(info common.ShellScriptInfo) string {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)

	// Set env variables from build script
	for _, keyValue := range b.GetVariables(info.Build, b.fullProjectDir(info), info.Environment) {
		b.executeCommand(w, "export", keyValue)
	}
	w.Flush()

	return buffer.String()
}

func (b *BashShell) generatePreBuildScript(info common.ShellScriptInfo) string {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)

	if len(info.Build.Hostname) != 0 {
		b.executeCommand(w, "echo", "Running on $(hostname) via " + info.Build.Hostname + "...")
	} else {
		b.executeCommand(w, "echo", "Running on $(hostname)...")
	}

	build := info.Build
	projectDir := b.fullProjectDir(info)
	gitDir := filepath.Join(projectDir, ".git")

	if build.AllowGitFetch {
		b.writeFetchCmd(w, build, projectDir, gitDir)
	} else {
		b.writeCloneCmd(w, build, projectDir)
	}

	b.writeCheckoutCmd(w, build)
	w.Flush()

	return buffer.String()
}

func (b *BashShell) generateCommands(info common.ShellScriptInfo) string {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)

	b.writeCdBuildDir(w, info)

	commands := info.Build.Commands
	commands = strings.TrimSpace(commands)
	for _, command := range strings.Split(commands, "\n") {
		command = strings.TrimSpace(command)
		if !helpers.BoolOrDefault(info.Build.Runner.DisableVerbose, false) {
			if command != "" {
				b.echoColored(w, "$ " + command)
			} else {
				b.executeCommand(w, "echo")
			}
		}
		io.WriteString(w, command+"\n")
	}

	w.Flush()

	return buffer.String()
}

func (b *BashShell) generatePostBuildScript(info common.ShellScriptInfo) string {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)
	b.writeCdBuildDir(w, info)

	w.Flush()

	return buffer.String()
}

func (b *BashShell) GenerateScript(info common.ShellScriptInfo) (*common.ShellScript, error) {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)

	io.WriteString(w, "#!/usr/bin/env bash\n\n")
	io.WriteString(w, b.generateExports(info))
	io.WriteString(w, "set -eo pipefail\n")
	io.WriteString(w, ": | eval " + helpers.ShellEscape(b.generatePreBuildScript(info)) + "\n")
	io.WriteString(w, "echo\n")
	io.WriteString(w, ": | eval " + helpers.ShellEscape(b.generateCommands(info)) + "\n")
	io.WriteString(w, "echo\n")
	io.WriteString(w, ": | eval " + helpers.ShellEscape(b.generatePostBuildScript(info)) + "\n")

	w.Flush()

	script := common.ShellScript{
		Script:      buffer.String(),
		Environment: b.GetVariables(info.Build, b.fullProjectDir(info), info.Environment),
	}

	// su
	if info.User != nil {
		script.Command = "su"
		if info.Type == common.LoginShell {
			script.Arguments = []string{"--shell", "/bin/bash", "--login", *info.User}
		} else {
			script.Arguments = []string{"--shell", "/bin/bash", *info.User}
		}
	} else {
		script.Command = "bash"
		if info.Type == common.LoginShell {
			script.Arguments = []string{"--login"}
		}
	}

	return &script, nil
}

func (b *BashShell) IsDefault() bool {
	return runtime.GOOS != "windows"
}

func init() {
	common.RegisterShell(&BashShell{})
}
