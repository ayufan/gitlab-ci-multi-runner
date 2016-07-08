package shells

import (
	"bufio"
	"bytes"
	"fmt"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"io"
	"path"
	"runtime"
	"strconv"
	"strings"
)

const bashDetectShell = `if [ -x /usr/local/bin/bash ]; then
	exec /usr/local/bin/bash $@
elif [ -x /usr/bin/bash ]; then
	exec /usr/bin/bash $@
elif [ -x /bin/bash ]; then
	exec /bin/bash $@
elif [ -x /usr/local/bin/sh ]; then
	exec /usr/local/bin/sh $@
elif [ -x /usr/bin/sh ]; then
	exec /usr/bin/sh $@
elif [ -x /bin/sh ]; then
	exec /bin/sh $@
else
	echo shell not found
	exit 1
fi

`

type BashShell struct {
	AbstractShell
	Shell string
}

type BashWriter struct {
	bytes.Buffer
	TemporaryPath string
	indent        int
}

func (b *BashWriter) Line(text string) {
	b.WriteString(strings.Repeat("  ", b.indent) + text + "\n")
}

func (b *BashWriter) CheckForErrors() {
}

func (b *BashWriter) Indent() {
	b.indent++
}

func (b *BashWriter) Unindent() {
	b.indent--
}

func (b *BashWriter) Command(command string, arguments ...string) {
	list := []string{
		helpers.ShellEscape(command),
	}

	for _, argument := range arguments {
		list = append(list, strconv.Quote(argument))
	}

	b.Line(strings.Join(list, " "))
}

func (b *BashWriter) Variable(variable common.BuildVariable) {
	if variable.File {
		variableFile := b.Absolute(path.Join(b.TemporaryPath, variable.Key))
		b.Line(fmt.Sprintf("mkdir -p %q", helpers.ToSlash(b.TemporaryPath)))
		b.Line(fmt.Sprintf("echo -n %s > %q", helpers.ShellEscape(variable.Value), variableFile))
		b.Line(fmt.Sprintf("export %s=%q", helpers.ShellEscape(variable.Key), variableFile))
	} else {
		b.Line(fmt.Sprintf("export %s=%s", helpers.ShellEscape(variable.Key), helpers.ShellEscape(variable.Value)))
	}
}

func (b *BashWriter) IfDirectory(path string) {
	b.Line(fmt.Sprintf("if [[ -d %q ]]; then", path))
	b.Indent()
}

func (b *BashWriter) IfFile(path string) {
	b.Line(fmt.Sprintf("if [[ -e %q ]]; then", path))
	b.Indent()
}

func (b *BashWriter) IfCmd(cmd string, arguments ...string) {
	b.Line(fmt.Sprintf("if %q %s >/dev/null 2>/dev/null; then", cmd, strings.Join(arguments, " ")))
	b.Indent()
}

func (b *BashWriter) Else() {
	b.Unindent()
	b.Line("else")
	b.Indent()
}

func (b *BashWriter) EndIf() {
	b.Unindent()
	b.Line("fi")
}

func (b *BashWriter) Cd(path string) {
	b.Command("cd", path)
}

func (b *BashWriter) RmDir(path string) {
	b.Command("rm", "-r", "-f", path)
}

func (b *BashWriter) RmFile(path string) {
	b.Command("rm", "-f", path)
}

func (b *BashWriter) Absolute(dir string) string {
	if path.IsAbs(dir) {
		return dir
	}
	return path.Join("$PWD", dir)
}

func (b *BashWriter) Print(format string, arguments ...interface{}) {
	coloredText := helpers.ANSI_RESET + fmt.Sprintf(format, arguments...)
	b.Line("echo " + helpers.ShellEscape(coloredText))
}

func (b *BashWriter) Notice(format string, arguments ...interface{}) {
	coloredText := helpers.ANSI_BOLD_GREEN + fmt.Sprintf(format, arguments...) + helpers.ANSI_RESET
	b.Line("echo " + helpers.ShellEscape(coloredText))
}

func (b *BashWriter) Warning(format string, arguments ...interface{}) {
	coloredText := helpers.ANSI_YELLOW + fmt.Sprintf(format, arguments...) + helpers.ANSI_RESET
	b.Line("echo " + helpers.ShellEscape(coloredText))
}

func (b *BashWriter) Error(format string, arguments ...interface{}) {
	coloredText := helpers.ANSI_BOLD_RED + fmt.Sprintf(format, arguments...) + helpers.ANSI_RESET
	b.Line("echo " + helpers.ShellEscape(coloredText))
}

func (b *BashWriter) EmptyLine() {
	b.Line("echo")
}

func (b *BashWriter) Finish() string {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)
	io.WriteString(w, "set -eo pipefail\n")
	io.WriteString(w, "set +o noclobber\n")
	io.WriteString(w, ": | eval "+helpers.ShellEscape(b.String())+"\n")
	w.Flush()
	return buffer.String()
}

func (b *BashShell) GetName() string {
	return b.Shell
}

func (b *BashShell) GetConfiguration(info common.ShellScriptInfo) (script *common.ShellConfiguration, err error) {
	var detectScript string
	var shellCommand string
	if info.Type == common.LoginShell {
		detectScript = strings.Replace(bashDetectShell, "$@", "--login", -1)
		shellCommand = b.Shell + " --login"
	} else {
		detectScript = strings.Replace(bashDetectShell, "$@", "", -1)
		shellCommand = b.Shell
	}

	script = &common.ShellConfiguration{}
	script.DockerCommand = []string{"sh", "-c", detectScript}

	// su
	if info.User != "" {
		script.Command = "su"
		if runtime.GOOS == "linux" {
			script.Arguments = append(script.Arguments, "-s", "/bin/"+b.Shell)
		}
		script.Arguments = append(script.Arguments, info.User)
		script.Arguments = append(script.Arguments, "-c", shellCommand)
	} else {
		script.Command = b.Shell
		if info.Type == common.LoginShell {
			script.Arguments = append(script.Arguments, "--login")
		}
	}

	return
}

func (b *BashShell) GenerateScript(scriptType common.ShellScriptType, info common.ShellScriptInfo) (script string, err error) {
	w := &BashWriter{
		TemporaryPath: info.Build.FullProjectDir() + ".tmp",
	}

	if scriptType == common.ShellPrepareScript {
		if len(info.Build.Hostname) != 0 {
			w.Line("echo " + strconv.Quote("Running on $(hostname) via "+info.Build.Hostname+"..."))
		} else {
			w.Line("echo " + strconv.Quote("Running on $(hostname)..."))
		}
	}

	err = b.writeScript(w, scriptType, info)
	script = w.Finish()
	return
}

func (b *BashShell) IsDefault() bool {
	return runtime.GOOS != "windows" && b.Shell == "bash"
}

func init() {
	common.RegisterShell(&BashShell{Shell: "sh"})
	common.RegisterShell(&BashShell{Shell: "bash"})
}
