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

type BashShell struct {
	AbstractShell
}

func (b *BashShell) GetName() string {
	return "bash"
}

func (b *BashShell) writeCloneCmd(w io.Writer, build *common.Build, projectDir string) {
	io.WriteString(w, "echo Cloning repository...\n")
	io.WriteString(w, fmt.Sprintf("rm -rf %s\n", projectDir))
	io.WriteString(w, fmt.Sprintf("mkdir -p %s\n", projectDir))
	io.WriteString(w, fmt.Sprintf("git clone %s %s\n", helpers.ShellEscape(build.RepoURL), projectDir))
	io.WriteString(w, fmt.Sprintf("cd %s\n", projectDir))
}

func (b *BashShell) writeFetchCmd(w io.Writer, build *common.Build, projectDir string, gitDir string) {
	io.WriteString(w, fmt.Sprintf("if [[ -d %s ]]; then\n", gitDir))
	io.WriteString(w, "echo Fetching changes...\n")
	io.WriteString(w, fmt.Sprintf("cd %s\n", projectDir))
	io.WriteString(w, fmt.Sprintf("git clean -fdx\n"))
	io.WriteString(w, fmt.Sprintf("git reset --hard > /dev/null\n"))
	io.WriteString(w, fmt.Sprintf("git remote set-url origin %s\n", helpers.ShellEscape(build.RepoURL)))
	io.WriteString(w, fmt.Sprintf("git fetch origin\n"))
	io.WriteString(w, fmt.Sprintf("else\n"))
	b.writeCloneCmd(w, build, projectDir)
	io.WriteString(w, fmt.Sprintf("fi\n"))
}

func (b *BashShell) writeCheckoutCmd(w io.Writer, build *common.Build) {
	io.WriteString(w, fmt.Sprintf("echo Checking out %s as %s...\n", build.Sha[0:8], build.RefName))
	io.WriteString(w, fmt.Sprintf("git checkout -qf %s\n", build.Sha))
}

func (b *BashShell) GenerateScript(info common.ShellScriptInfo) (*common.ShellScript, error) {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)

	build := info.Build
	projectDir := build.FullProjectDir()
	projectDir = helpers.ToSlash(projectDir)
	gitDir := filepath.Join(projectDir, ".git")

	projectScript := helpers.ShellEscape(build.FullProjectDir() + ".sh")

	io.WriteString(w, "#!/usr/bin/env bash\n")
	io.WriteString(w, "\n")
	io.WriteString(w, "set -eo pipefail\n")
	io.WriteString(w, "\n")
	io.WriteString(w, "# save script that is read from to file and execute script file on remote server\n")
	io.WriteString(w, fmt.Sprintf("mkdir -p %s\n", helpers.ShellEscape(projectDir)))
	io.WriteString(w, fmt.Sprintf("cat > %s; source %s\n", projectScript, projectScript))
	io.WriteString(w, "\n")
	if len(build.Hostname) != 0 {
		io.WriteString(w, fmt.Sprintf("echo Running on $(hostname) via %s...\n", helpers.ShellEscape(build.Hostname)))
	} else {
		io.WriteString(w, "echo Running on $(hostname)...\n")
	}

	io.WriteString(w, "\n")
	if build.AllowGitFetch {
		b.writeFetchCmd(w, build, helpers.ShellEscape(projectDir), helpers.ShellEscape(gitDir))
	} else {
		b.writeCloneCmd(w, build, helpers.ShellEscape(projectDir))
	}

	b.writeCheckoutCmd(w, build)
	io.WriteString(w, "\n")
	if !helpers.BoolOrDefault(build.Runner.DisableVerbose, false) {
		io.WriteString(w, "set -v\n")
		io.WriteString(w, "\n")
	}

	commands := build.Commands
	commands = strings.Replace(commands, "\r\n", "\n", -1)
	io.WriteString(w, commands)
	io.WriteString(w, "\n")

	w.Flush()

	script := common.ShellScript{
		Environment: b.GetVariables(build, projectDir),
		Script:      buffer.String(),
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
