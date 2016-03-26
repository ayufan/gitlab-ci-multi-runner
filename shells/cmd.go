package shells

import (
	"bufio"
	"bytes"
	"fmt"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"io"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

type CmdShell struct {
	AbstractShell
}

type CmdWriter struct {
	bytes.Buffer
	TemporaryPath string
	indent        int
}

func batchQuote(text string) string {
	return "\"" + batchEscape(text) + "\""
}

func batchEscape(text string) string {
	// taken from: http://www.robvanderwoude.com/escapechars.php
	text = strings.Replace(text, "^", "^^", -1)
	text = strings.Replace(text, "!", "^^!", -1)
	text = strings.Replace(text, "&", "^&", -1)
	text = strings.Replace(text, "<", "^<", -1)
	text = strings.Replace(text, ">", "^>", -1)
	text = strings.Replace(text, "|", "^|", -1)
	text = strings.Replace(text, "\r", "", -1)
	text = strings.Replace(text, "\n", "!nl!", -1)
	return text
}

func batchEscapeVariable(text string) string {
	text = strings.Replace(text, "%", "%%", -1)
	text = batchEscape(text)
	return text
}

func (b *CmdShell) GetName() string {
	return "cmd"
}

func (b *CmdWriter) Line(text string) {
	b.WriteString(strings.Repeat("  ", b.indent) + text + "\r\n")
}

func (b *CmdWriter) CheckForErrors() {
	b.checkErrorLevel()
}

func (b *CmdWriter) Indent() {
	b.indent++
}

func (b *CmdWriter) Unindent() {
	b.indent--
}

func (b *CmdWriter) checkErrorLevel() {
	b.Line("IF %errorlevel% NEQ 0 exit /b %errorlevel%")
	b.Line("")
}

func (b *CmdWriter) Command(command string, arguments ...string) {
	list := []string{
		batchQuote(command),
	}

	for _, argument := range arguments {
		list = append(list, batchQuote(argument))
	}

	b.Line(strings.Join(list, " "))
	b.checkErrorLevel()
}

func (b *CmdWriter) Variable(variable common.BuildVariable) {
	if variable.File {
		variableFile := b.Absolute(path.Join(b.TemporaryPath, variable.Key))
		variableFile = helpers.ToBackslash(variableFile)
		b.Line(fmt.Sprintf("md %q 2>NUL 1>NUL", batchEscape(helpers.ToBackslash(b.TemporaryPath))))
		b.Line(fmt.Sprintf("echo %s > %s", batchEscapeVariable(variable.Value), batchEscape(variableFile)))
		b.Line("SET " + batchEscapeVariable(variable.Key) + "=" + batchEscape(variableFile))
	} else {
		b.Line("SET " + batchEscapeVariable(variable.Key) + "=" + batchEscapeVariable(variable.Value))
	}
}

func (b *CmdWriter) IfDirectory(path string) {
	b.Line("IF EXIST " + batchQuote(helpers.ToBackslash(path)) + " (")
	b.Indent()
}

func (b *CmdWriter) IfFile(path string) {
	b.Line("IF EXIST " + batchQuote(helpers.ToBackslash(path)) + " (")
	b.Indent()
}

func (b *CmdWriter) Else() {
	b.Unindent()
	b.Line(") ELSE (")
	b.Indent()
}

func (b *CmdWriter) EndIf() {
	b.Unindent()
	b.Line(")")
}

func (b *CmdWriter) Cd(path string) {
	b.Line("cd /D " + batchQuote(helpers.ToBackslash(path)))
	b.checkErrorLevel()
}

func (b *CmdWriter) RmDir(path string) {
	b.Line("rd /s /q " + batchQuote(helpers.ToBackslash(path)) + " 2>NUL 1>NUL")
}

func (b *CmdWriter) RmFile(path string) {
	b.Line("rd /s /q " + batchQuote(helpers.ToBackslash(path)) + " 2>NUL 1>NUL")
}

func (b *CmdWriter) Print(format string, arguments ...interface{}) {
	coloredText := fmt.Sprintf(format, arguments...)
	b.Line("echo " + batchEscapeVariable(coloredText))
}

func (b *CmdWriter) Notice(format string, arguments ...interface{}) {
	coloredText := fmt.Sprintf(format, arguments...)
	b.Line("echo " + batchEscapeVariable(coloredText))
}

func (b *CmdWriter) Warning(format string, arguments ...interface{}) {
	coloredText := fmt.Sprintf(format, arguments...)
	b.Line("echo " + batchEscapeVariable(coloredText))
}

func (b *CmdWriter) Error(format string, arguments ...interface{}) {
	coloredText := fmt.Sprintf(format, arguments...)
	b.Line("echo " + batchEscapeVariable(coloredText))
}

func (b *CmdWriter) EmptyLine() {
	b.Line("echo.")
}

func (b *CmdWriter) Absolute(dir string) string {
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join("%CD%", dir)
}

func (b *CmdWriter) GetScript() string {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)
	io.WriteString(w, "@echo off\r\n")
	io.WriteString(w, "setlocal enableextensions\r\n")
	io.WriteString(w, "setlocal enableDelayedExpansion\r\n")
	io.WriteString(w, "set nl=^\r\n\r\n\r\n")
	io.WriteString(w, b.String())
	w.Flush()
	return buffer.String()
}

func (b *CmdWriter) GetCommand(login bool) (cmd string, args []string) {
	return "cmd", []string{"/Q", "/C"}
}

func (b *CmdShell) shell(build *common.Build, handler func(w ShellWriter) error) (script *common.ShellScript, err error) {
	temporaryPath := build.FullProjectDir() + ".tmp"
	w := &CmdWriter{TemporaryPath: temporaryPath}
	handler(w)
	script = &common.ShellScript{
		Command:   "cmd",
		Arguments: []string{"/Q", "/C"},
		Extension: "cmd",
		Script:    w.GetScript(),
	}
	return
}

func (b *CmdShell) PreBuild(build *common.Build, options common.BuildOptions) (script *common.ShellScript, err error) {
	return b.shell(build, func(w ShellWriter) error {
		if len(build.Hostname) != 0 {
			w.Line("echo Running on %COMPUTERNAME% via " + batchEscape(build.Hostname) + "...")
		} else {
			w.Line("echo Running on %COMPUTERNAME%...")
		}
		w.Line("")
		b.GeneratePreBuild(w, build)
		return nil
	})
}

func (b *CmdShell) Build(build *common.Build, options common.BuildOptions) (script *common.ShellScript, err error) {
	return b.shell(build, func(w ShellWriter) error {
		b.GenerateCommands(w, build)
		return nil
	})
}

func (b *CmdShell) PostBuild(build *common.Build, options common.BuildOptions) (script *common.ShellScript, err error) {
	return b.shell(build, func(w ShellWriter) error {
		b.GeneratePostBuild(w, build)
		return nil
	})
}

func (b *CmdShell) IsDefault() bool {
	return runtime.GOOS == "windows"
}

func init() {
	common.RegisterShell(&CmdShell{})
}
