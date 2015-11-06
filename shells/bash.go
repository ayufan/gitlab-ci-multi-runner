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
)

type BashShell struct {
	AbstractShell
}

func (b *BashShell) GetName() string {
	return "bash"
}

func (b *BashShell) echoColored(w io.Writer, text string) {
	coloredText := helpers.ANSI_BOLD_GREEN + text + helpers.ANSI_RESET
	io.WriteString(w, "echo " + helpers.ShellEscape(coloredText) + "\n")
}

func (b *BashShell) echoWarning(w io.Writer, text string) {
	coloredText := helpers.ANSI_BOLD_YELLOW + text + helpers.ANSI_RESET
	io.WriteString(w, "echo " + helpers.ShellEscape(coloredText) + "\n")
}

func (b *BashShell) echoColoredFormat(w io.Writer, format string, a ...interface{}) {
	b.echoColored(w, fmt.Sprintf(format, a...))
}

func (b *BashShell) executeCommand(w io.Writer, cmd string, arguments ...string) {
	list := []string{
		helpers.ShellEscape(cmd),
	}

	for _, argument := range arguments {
		list = append(list, helpers.ShellEscape(argument))
	}

	io.WriteString(w, strings.Join(list, " ") + "\n")
}

func (b *BashShell) writeCloneCmd(w io.Writer, build *common.Build, projectDir string) {
	b.echoColoredFormat(w, "Cloning repository...")
	io.WriteString(w, fmt.Sprintf("rm -rf %s\n", projectDir))
	io.WriteString(w, fmt.Sprintf("mkdir -p %s\n", projectDir))
	io.WriteString(w, fmt.Sprintf("git clone %s %s\n", helpers.ShellEscape(build.RepoURL), projectDir))
	io.WriteString(w, fmt.Sprintf("cd %s\n", projectDir))
}

func (b *BashShell) writeFetchCmd(w io.Writer, build *common.Build, projectDir string, gitDir string) {
	io.WriteString(w, fmt.Sprintf("if [[ -d %s ]]; then\n", gitDir))
	b.echoColoredFormat(w, "Fetching changes...")
	io.WriteString(w, fmt.Sprintf("cd %s\n", projectDir))
	io.WriteString(w, fmt.Sprintf("git clean -ffdx\n"))
	io.WriteString(w, fmt.Sprintf("git reset --hard > /dev/null\n"))
	io.WriteString(w, fmt.Sprintf("git remote set-url origin %s\n", helpers.ShellEscape(build.RepoURL)))
	io.WriteString(w, fmt.Sprintf("git fetch origin\n"))
	io.WriteString(w, fmt.Sprintf("else\n"))
	b.writeCloneCmd(w, build, projectDir)
	io.WriteString(w, fmt.Sprintf("fi\n"))
}

func (b *BashShell) writeCheckoutCmd(w io.Writer, build *common.Build) {
	b.echoColoredFormat(w, "Checking out %s as %s...", build.Sha[0:8], build.RefName)
	io.WriteString(w, fmt.Sprintf("git checkout -qf %s\n", build.Sha))
}

func (b *BashShell) writeCd(w io.Writer, info common.ShellScriptInfo) {
	io.WriteString(w, fmt.Sprintf("cd %s\n", helpers.ShellEscape(b.fullProjectDir(info))))
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
		io.WriteString(w, "export " + helpers.ShellEscape(keyValue) + "\n")
	}
	w.Flush()

	return buffer.String()
}

func (b *BashShell) generateCloneScript(info common.ShellScriptInfo) string {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)

	build := info.Build
	projectDir := b.fullProjectDir(info)
	gitDir := filepath.Join(projectDir, ".git")

	if build.AllowGitFetch {
		b.writeFetchCmd(w, build, helpers.ShellEscape(projectDir), helpers.ShellEscape(gitDir))
	} else {
		b.writeCloneCmd(w, build, helpers.ShellEscape(projectDir))
	}

	b.writeCheckoutCmd(w, build)
	w.Flush()

	return buffer.String()
}

func (b *BashShell) generateCommands(info common.ShellScriptInfo) string {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)

	b.writeCd(w, info)

	commands := info.Build.Commands
	commands = strings.TrimSpace(commands)
	for _, command := range strings.Split(commands, "\n") {
		command = strings.TrimSpace(command)
		if !helpers.BoolOrDefault(info.Build.Runner.DisableVerbose, false) {
			if command != "" {
				b.echoColored(w, "$ " + command)
			} else {
				io.WriteString(w, "echo\n")
			}
		}
		io.WriteString(w, command+"\n")
	}

	w.Flush()

	return buffer.String()
}

func (b *BashShell) GenerateScript(info common.ShellScriptInfo) (*common.ShellScript, error) {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)

	io.WriteString(w, "#!/usr/bin/env bash\n\n")
	if len(info.Build.Hostname) != 0 {
		io.WriteString(w, fmt.Sprintf("echo Running on $(hostname) via %s...", helpers.ShellEscape(info.Build.Hostname)))
	} else {
		io.WriteString(w, "echo Running on $(hostname)...\n")
	}
	io.WriteString(w, b.generateExports(info))
	io.WriteString(w, "set -eo pipefail\n")
	io.WriteString(w, ": | eval " + helpers.ShellEscape(b.generateCloneScript(info)) + "\n")
	io.WriteString(w, "echo\n")
	io.WriteString(w, ": | eval " + helpers.ShellEscape(b.generateCommands(info)) + "\n")

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
