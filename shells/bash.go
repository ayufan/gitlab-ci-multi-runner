package shells

import (
	"bufio"
	"bytes"
	"fmt"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"io"
	"runtime"
	"strings"
)

type BashShell struct {
}

func (b *BashShell) GetName() string {
	return "bash"
}

func (b *BashShell) writeCloneCmd(w io.Writer, build *common.Build) {
	io.WriteString(w, "echo Clonning repository...\n")
	io.WriteString(w, fmt.Sprintf("rm -rf %s\n", build.FullProjectDir()))
	io.WriteString(w, fmt.Sprintf("mkdir -p %s\n", build.FullProjectDir()))
	io.WriteString(w, fmt.Sprintf("git clone %s %s\n", build.RepoURL, build.FullProjectDir()))
	io.WriteString(w, fmt.Sprintf("cd %s\n", build.FullProjectDir()))
}

func (b *BashShell) writeFetchCmd(w io.Writer, build *common.Build) {
	io.WriteString(w, fmt.Sprintf("if [[ -d %s/.git ]]; then\n", build.FullProjectDir()))
	io.WriteString(w, "echo Fetching changes...\n")
	io.WriteString(w, fmt.Sprintf("cd %s\n", build.FullProjectDir()))
	io.WriteString(w, fmt.Sprintf("git clean -fdx\n"))
	io.WriteString(w, fmt.Sprintf("git reset --hard > /dev/null\n"))
	io.WriteString(w, fmt.Sprintf("git remote set-url origin %s\n", build.RepoURL))
	io.WriteString(w, fmt.Sprintf("git fetch origin\n"))
	io.WriteString(w, fmt.Sprintf("else\n"))
	b.writeCloneCmd(w, build)
	io.WriteString(w, fmt.Sprintf("fi\n"))
}

func (b *BashShell) writeCheckoutCmd(w io.Writer, build *common.Build) {
	io.WriteString(w, fmt.Sprintf("echo Checkouting %s as %s...\n", build.Sha[0:8], build.RefName))
	io.WriteString(w, fmt.Sprintf("git checkout -qf %s\n", build.Sha))
}

func (b *BashShell) GenerateScript(build *common.Build, shellType common.ShellType) (*common.ShellScript, error) {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)

	io.WriteString(w, "#!/usr/bin/env bash\n")
	io.WriteString(w, "\n")
	if len(build.Hostname) != 0 {
		io.WriteString(w, fmt.Sprintf("echo Running on $(hostname) via %s...\n", helpers.ShellEscape(build.Hostname)))
	} else {
		io.WriteString(w, "echo Running on $(hostname)...\n")
	}
	io.WriteString(w, "\n")
	io.WriteString(w, "set -eo pipefail\n")

	io.WriteString(w, "\n")
	if build.AllowGitFetch {
		b.writeFetchCmd(w, build)
	} else {
		b.writeCloneCmd(w, build)
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

	w.Flush()

	env := []string{
		fmt.Sprintf("CI_BUILD_REF=%s", build.Sha),
		fmt.Sprintf("CI_BUILD_BEFORE_SHA=%s", build.BeforeSha),
		fmt.Sprintf("CI_BUILD_REF_NAME=%s", build.RefName),
		fmt.Sprintf("CI_BUILD_ID=%d", build.ID),
		fmt.Sprintf("CI_BUILD_REPO=%s", build.RepoURL),

		fmt.Sprintf("CI_PROJECT_ID=%d", build.ProjectID),
		fmt.Sprintf("CI_PROJECT_DIR=%s", build.FullProjectDir()),

		"CI_SERVER=yes",
		"CI_SERVER_NAME=GitLab CI",
		"CI_SERVER_VERSION=",
		"CI_SERVER_REVISION=",
	}

	script := common.ShellScript{
		Environment: env,
		Script:      buffer.Bytes(),
		Command:     "bash",
	}

	if shellType == common.LoginShell {
		script.Arguments = []string{"--login"}
	}

	return &script, nil
}

func (b *BashShell) IsDefault() bool {
	return runtime.GOOS != "windows"
}

func init() {
	common.RegisterShell(&BashShell{})
}
