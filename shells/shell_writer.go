package shells

import "gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"

type ShellWriter interface {
	Variable(variable common.BuildVariable)
	Command(command string, arguments ...string)
	Line(text string)
	CheckForErrors()

	IfDirectory(path string)
	IfFile(file string)
	Else()
	EndIf()

	Cd(path string)
	RmDir(path string)
	RmFile(path string)
	Absolute(path string) string

	Print(fmt string, arguments ...interface{})
	Notice(fmt string, arguments ...interface{})
	Warning(fmt string, arguments ...interface{})
	Error(fmt string, arguments ...interface{})
	EmptyLine()
}
