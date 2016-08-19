package common

import "github.com/stretchr/testify/mock"

type MockShell struct {
	mock.Mock
}

func (m *MockShell) GetName() string {
	ret := m.Called()

	r0 := ret.Get(0).(string)

	return r0
}
func (m *MockShell) GetSupportedOptions() []string {
	ret := m.Called()

	var r0 []string
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]string)
	}

	return r0
}
func (m *MockShell) GetFeatures(features *FeaturesInfo) {
	m.Called(features)
}
func (m *MockShell) IsDefault() bool {
	ret := m.Called()

	r0 := ret.Get(0).(bool)

	return r0
}
func (m *MockShell) GetConfiguration(info ShellScriptInfo) (*ShellConfiguration, error) {
	ret := m.Called(info)

	var r0 *ShellConfiguration
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*ShellConfiguration)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *MockShell) GenerateScript(scriptType ShellScriptType, info ShellScriptInfo) (string, error) {
	ret := m.Called(scriptType, info)

	r0 := ret.Get(0).(string)
	r1 := ret.Error(1)

	return r0, r1
}
