package mocks

import "gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
import "github.com/stretchr/testify/mock"

type Shell struct {
	mock.Mock
}

func (m *Shell) GetName() string {
	ret := m.Called()

	r0 := ret.Get(0).(string)

	return r0
}
func (m *Shell) GetSupportedOptions() []string {
	ret := m.Called()

	var r0 []string
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]string)
	}

	return r0
}
func (m *Shell) GenerateScript(info common.ShellScriptInfo) (*common.ShellScript, error) {
	ret := m.Called(info)

	var r0 *common.ShellScript
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.ShellScript)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Shell) GetFeatures(features *common.FeaturesInfo) {
	m.Called(features)
}
func (m *Shell) IsDefault() bool {
	ret := m.Called()

	r0 := ret.Get(0).(bool)

	return r0
}
