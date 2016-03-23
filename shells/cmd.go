package shells

import (
	"bytes"
	"fmt"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
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

func (b *CmdShell) GenerateScript(info common.ShellScriptInfo) (*common.ShellScript, error) {
	w := &CmdWriter{
		TemporaryPath: info.Build.FullProjectDir() + ".tmp",
	}
	w.Line("@echo off")
	w.Line("setlocal enableextensions")
	w.Line("setlocal enableDelayedExpansion")
	w.Line("set nl=^\r\n\r\n")

	if len(info.Build.Hostname) != 0 {
		w.Line("echo Running on %COMPUTERNAME% via " + batchEscape(info.Build.Hostname) + "...")
	} else {
		w.Line("echo Running on %COMPUTERNAME%...")
	}
	w.Line("")

	w.Line("call :prescript")
	w.checkErrorLevel()
	w.Line("call :buildscript")
	w.checkErrorLevel()
	w.Line("call :postscript")
	w.checkErrorLevel()
	w.Line("goto :EOF")

	w.Line(":prescript")
	b.GeneratePreBuild(w, info)
	w.Line("goto :EOF")
	w.Line("")

	w.Line(":buildscript")
	b.GenerateCommands(w, info)
	w.Line("goto :EOF")
	w.Line("")

	w.Line(":postscript")
	b.GeneratePostBuild(w, info)
	w.Line("goto :EOF")
	w.Line("")

	script := common.ShellScript{
		BuildScript: w.String(),
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
