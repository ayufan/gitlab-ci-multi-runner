package mocks

import "gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
import "github.com/stretchr/testify/mock"

type Executor struct {
	mock.Mock
}

func (m *Executor) Shell() *common.ShellScriptInfo {
	ret := m.Called()

	var r0 *common.ShellScriptInfo
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.ShellScriptInfo)
	}

	return r0
}
func (m *Executor) Prepare(globalConfig *common.Config, config *common.RunnerConfig, build *common.Build) error {
	ret := m.Called(globalConfig, config, build)

	r0 := ret.Error(0)

	return r0
}
func (m *Executor) Run(cmd common.ExecutorCommand) error {
	ret := m.Called(cmd)

	r0 := ret.Error(0)

	return r0
}
func (m *Executor) Finish(err error) {
	m.Called(err)
}
func (m *Executor) Cleanup() {
	m.Called()
}
