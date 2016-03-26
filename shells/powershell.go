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
	"strings"
)

type PowerShell struct {
	AbstractShell
}

type PsWriter struct {
	bytes.Buffer
	TemporaryPath string
	indent        int
}

func psQuote(text string) string {
	// taken from: http://www.robvanderwoude.com/escapechars.php
	text = strings.Replace(text, "`", "``", -1)
	// text = strings.Replace(text, "\0", "`0", -1)
	text = strings.Replace(text, "\a", "`a", -1)
	text = strings.Replace(text, "\b", "`b", -1)
	text = strings.Replace(text, "\f", "^f", -1)
	text = strings.Replace(text, "\r", "`r", -1)
	text = strings.Replace(text, "\n", "`n", -1)
	text = strings.Replace(text, "\t", "^t", -1)
	text = strings.Replace(text, "\v", "^v", -1)
	text = strings.Replace(text, "#", "`#", -1)
	text = strings.Replace(text, "'", "`'", -1)
	text = strings.Replace(text, "\"", "`\"", -1)
	return "\"" + text + "\""
}

func psQuoteVariable(text string) string {
	text = psQuote(text)
	text = strings.Replace(text, "$", "`$", -1)
	return text
}

func (b *PsWriter) Line(text string) {
	b.WriteString(strings.Repeat("  ", b.indent) + text + "\r\n")
}

func (b *PsWriter) CheckForErrors() {
	b.checkErrorLevel()
}

func (b *PsWriter) Indent() {
	b.indent++
}

func (b *PsWriter) Unindent() {
	b.indent--
}

func (b *PsWriter) checkErrorLevel() {
	b.Line("if(!$?) { Exit $LASTEXITCODE }")
	b.Line("")
}

func (b *PsWriter) Command(command string, arguments ...string) {
	list := []string{
		psQuote(command),
	}

	for _, argument := range arguments {
		list = append(list, psQuote(argument))
	}

	b.Line("& " + strings.Join(list, " "))
	b.checkErrorLevel()
}

func (b *PsWriter) Variable(variable common.BuildVariable) {
	if variable.File {
		variableFile := b.Absolute(path.Join(b.TemporaryPath, variable.Key))
		variableFile = helpers.ToBackslash(variableFile)
		b.Line(fmt.Sprintf("md %s -Force | out-null", psQuote(helpers.ToBackslash(b.TemporaryPath))))
		b.Line(fmt.Sprintf("Set-Content %s -Value %s -Encoding UTF8 -Force", psQuote(variableFile), psQuoteVariable(variable.Value)))
		b.Line("$" + variable.Key + "=" + psQuote(variableFile))
	} else {
		b.Line("$" + variable.Key + "=" + psQuoteVariable(variable.Value))
	}

	b.Line("$env:" + variable.Key + "=$" + variable.Key)
}

func (b *PsWriter) IfDirectory(path string) {
	b.Line("if(Test-Path " + psQuote(helpers.ToBackslash(path)) + " -PathType Container) {")
	b.Indent()
}

func (b *PsWriter) IfFile(path string) {
	b.Line("if(Test-Path " + psQuote(helpers.ToBackslash(path)) + " -PathType Leaf) {")
	b.Indent()
}

func (b *PsWriter) Else() {
	b.Unindent()
	b.Line("} else {")
	b.Indent()
}

func (b *PsWriter) EndIf() {
	b.Unindent()
	b.Line("}")
}

func (b *PsWriter) Cd(path string) {
	b.Line("cd " + psQuote(helpers.ToBackslash(path)))
	b.checkErrorLevel()
}

func (b *PsWriter) RmDir(path string) {
	path = psQuote(helpers.ToBackslash(path))
	b.Line("if( (Get-Command -Name Remove-Item2 -Module NTFSSecurity -ErrorAction SilentlyContinue) -and (Test-Path " + path + " -PathType Container) ) {")
	b.Indent()
	b.Line("Remove-Item2 -Force -Recurse " + path)
	b.Unindent()
	b.Line("} elseif(Test-Path " + path + ") {")
	b.Indent()
	b.Line("Remove-Item -Force -Recurse " + path)
	b.Unindent()
	b.Line("}")
	b.Line("")
}

func (b *PsWriter) RmFile(path string) {
	path = psQuote(helpers.ToBackslash(path))
	b.Line("if( (Get-Command -Name Remove-Item2 -Module NTFSSecurity -ErrorAction SilentlyContinue) -and (Test-Path " + path + " -PathType Leaf) ) {")
	b.Indent()
	b.Line("Remove-Item2 -Force " + path)
	b.Unindent()
	b.Line("} elseif(Test-Path " + path + ") {")
	b.Indent()
	b.Line("Remove-Item -Force " + path)
	b.Unindent()
	b.Line("}")
	b.Line("")
}

func (b *PsWriter) Print(format string, arguments ...interface{}) {
	coloredText := fmt.Sprintf(format, arguments...)
	b.Line("echo " + psQuoteVariable(coloredText))
}

func (b *PsWriter) Notice(format string, arguments ...interface{}) {
	coloredText := fmt.Sprintf(format, arguments...)
	b.Line("echo " + psQuoteVariable(coloredText))
}

func (b *PsWriter) Warning(format string, arguments ...interface{}) {
	coloredText := fmt.Sprintf(format, arguments...)
	b.Line("echo " + psQuoteVariable(coloredText))
}

func (b *PsWriter) Error(format string, arguments ...interface{}) {
	coloredText := fmt.Sprintf(format, arguments...)
	b.Line("echo " + psQuoteVariable(coloredText))
}

func (b *PsWriter) EmptyLine() {
	b.Line("echo \"\"")
}

func (b *PsWriter) Absolute(dir string) string {
	if filepath.IsAbs(dir) {
		return dir
	}

	b.Line("$CurrentDirectory = (Resolve-Path .\\).Path")
	return filepath.Join("$CurrentDirectory", dir)
}

func (b *PowerShell) GetName() string {
	return "powershell"
}

func (b *PsWriter) GetScript() string {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)
	io.WriteString(w, "$ErrorActionPreference = \"Stop\"\r\n")
	io.WriteString(w, b.String())
	w.Flush()
	return buffer.String()
}

func (b *PsWriter) GetCommand(login bool) (cmd string, args []string) {
	return "powershell", []string{"-noprofile", "-noninteractive", "-executionpolicy", "Bypass", "-command"}
}

func (b *PowerShell) shell(build *common.Build, handler func(w ShellWriter) error) (script *common.ShellScript, err error) {
	temporaryPath := build.FullProjectDir() + ".tmp"
	w := &CmdWriter{TemporaryPath: temporaryPath}
	handler(w)
	script = &common.ShellScript{
		Command:   "powershell",
		Arguments: []string{"-noprofile", "-noninteractive", "-executionpolicy", "Bypass", "-command"},
		Extension: "ps1",
		Script:    w.GetScript(),
	}
	return
}

func (b *PowerShell) PreBuild(build *common.Build, options common.BuildOptions) (script *common.ShellScript, err error) {
	return b.shell(build, func(w ShellWriter) error {
		if len(build.Hostname) != 0 {
			w.Line("echo \"Running on $env:computername via " + psQuoteVariable(build.Hostname) + "...\"")
		} else {
			w.Line("echo \"Running on $env:computername...\"")
		}
		w.Line("")
		b.GeneratePreBuild(w, build)
		return nil
	})
}

func (b *PowerShell) Build(build *common.Build, options common.BuildOptions) (script *common.ShellScript, err error) {
	return b.shell(build, func(w ShellWriter) error {
		b.GenerateCommands(w, build, options)
		return nil
	})
}

func (b *PowerShell) PostBuild(build *common.Build, options common.BuildOptions) (script *common.ShellScript, err error) {
	return b.shell(build, func(w ShellWriter) error {
		b.GeneratePostBuild(w, build)
		return nil
	})
}

func (b *PowerShell) IsDefault() bool {
	return false
}

func init() {
	common.RegisterShell(&PowerShell{})
}
