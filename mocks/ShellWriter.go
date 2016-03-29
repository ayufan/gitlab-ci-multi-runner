package mocks

import "github.com/stretchr/testify/mock"

import "gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"

type ShellWriter struct {
	mock.Mock
}

func (m *ShellWriter) Variable(variable common.BuildVariable) {
	m.Called(variable)
}
func (m *ShellWriter) Command(command string, arguments ...string) {
	m.Called(command, arguments)
}
func (m *ShellWriter) Line(text string) {
	m.Called(text)
}
func (m *ShellWriter) CheckForErrors() {
	m.Called()
}
func (m *ShellWriter) IfDirectory(path string) {
	m.Called(path)
}
func (m *ShellWriter) IfFile(file string) {
	m.Called(file)
}
func (m *ShellWriter) Else() {
	m.Called()
}
func (m *ShellWriter) EndIf() {
	m.Called()
}
func (m *ShellWriter) Cd(path string) {
	m.Called(path)
}
func (m *ShellWriter) RmDir(path string) {
	m.Called(path)
}
func (m *ShellWriter) RmFile(path string) {
	m.Called(path)
}
func (m *ShellWriter) Absolute(path string) string {
	ret := m.Called(path)

	r0 := ret.Get(0).(string)

	return r0
}
func (m *ShellWriter) Print(fmt string, arguments ...interface{}) {
	m.Called(fmt, arguments)
}
func (m *ShellWriter) Notice(fmt string, arguments ...interface{}) {
	m.Called(fmt, arguments)
}
func (m *ShellWriter) Warning(fmt string, arguments ...interface{}) {
	m.Called(fmt, arguments)
}
func (m *ShellWriter) Error(fmt string, arguments ...interface{}) {
	m.Called(fmt, arguments)
}
func (m *ShellWriter) EmptyLine() {
	m.Called()
}
