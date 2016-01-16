package shells

import (
	"bytes"
	"fmt"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"strings"
)

type PowerShell struct {
	AbstractShell
}

type PsWriter struct {
	bytes.Buffer
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
	text = strings.Replace(text, "$", "`$", -1)
	text = psQuote(text)
	return text
}

func (b *PsWriter) Line(text string) {
	b.WriteString(text + "\r\n")
}

func (b *PsWriter) checkErrorLevel() {
	b.Line("if (!$?) { Exit $LASTEXITCODE }")
}

func (b *PsWriter) Command(command string, arguments ...string) {
	list := []string{
		psQuote(command),
	}

	for _, argument := range arguments {
		list = append(list, psQuote(argument))
	}

	b.Line(strings.Join(list, " "))
	b.checkErrorLevel()
}

func (b *PsWriter) Variable(variable common.BuildVariable) {
	b.Line("$env:" + variable.Key + "=" + psQuoteVariable(variable.Value))
}

func (b *PsWriter) IfDirectory(path string) {
	b.Line("if(Test-Path " + psQuote(helpers.ToBackslash(path)) + " {")
}

func (b *PsWriter) IfFile(path string) {
	b.Line("if(Test-Path " + psQuote(helpers.ToBackslash(path)) + " {")
}

func (b *PsWriter) Else() {
	b.Line("} else {")
}

func (b *PsWriter) EndIf() {
	b.Line("}")
}

func (b *PsWriter) Cd(path string) {
	b.Line("cd " + psQuote(helpers.ToBackslash(path)))
	b.checkErrorLevel()
}

func (b *PsWriter) RmDir(path string) {
	path = psQuote(helpers.ToBackslash(path))
	b.Line("if( (Get-Command -Name Remove-Item2 -Module NTFSSecurity -ErrorAction SilentlyContinue) -and (Test-Path " + path + ") ) {")
	b.Line("Remove-Item2 -Force -Recurse " + path)
	b.Line("} elseif(Test-Path " + path + ") {")
	b.Line("Remove-Item -Force -Recurse " + path)
	b.Line("}")
}

func (b *PsWriter) RmFile(path string) {
	path = psQuote(helpers.ToBackslash(path))
	b.Line("if( (Get-Command -Name Remove-Item2 -Module NTFSSecurity -ErrorAction SilentlyContinue) -and (Test-Path " + path + ") ) {")
	b.Line("Remove-Item2 -Force " + path)
	b.Line("} elseif(Test-Path " + path + ") {")
	b.Line("Remove-Item -Force " + path)
	b.Line("}")
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

func (b *PowerShell) GetName() string {
	return "powershell"
}

func (b *PowerShell) GenerateScript(info common.ShellScriptInfo) (*common.ShellScript, error) {
	w := &PsWriter{}
	w.Line("$ErrorActionPreference = \"Stop\"")
	w.EmptyLine()

	if len(info.Build.Hostname) != 0 {
		w.Line("echo Running on $env:computername via " + psQuoteVariable(info.Build.Hostname) + "...")
	} else {
		w.Line("echo Running on  $env:computername...")
	}

	w.Line("& {")
	b.GeneratePreBuild(w, info)
	w.Line("}")
	w.checkErrorLevel()

	w.Line("& {")
	b.GenerateCommands(w, info)
	w.Line("}")
	w.checkErrorLevel()

	w.Line("& {")
	b.GeneratePostBuild(w, info)
	w.Line("}")
	w.checkErrorLevel()

	script := common.ShellScript{
		BuildScript: w.String(),
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
