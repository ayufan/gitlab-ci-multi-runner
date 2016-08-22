package common

import "github.com/stretchr/testify/mock"

type MockExecutorProvider struct {
	mock.Mock
}

func (m *MockExecutorProvider) CanCreate() bool {
	ret := m.Called()

	r0 := ret.Get(0).(bool)

	return r0
}
func (m *MockExecutorProvider) Create() Executor {
	ret := m.Called()

	r0 := ret.Get(0).(Executor)

	return r0
}
func (m *MockExecutorProvider) Acquire(config *RunnerConfig) (ExecutorData, error) {
	ret := m.Called(config)

	r0 := ret.Get(0).(ExecutorData)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *MockExecutorProvider) Release(config *RunnerConfig, data ExecutorData) error {
	ret := m.Called(config, data)

	r0 := ret.Error(0)

	return r0
}
func (m *MockExecutorProvider) GetFeatures(features *FeaturesInfo) {
	m.Called(features)
}
