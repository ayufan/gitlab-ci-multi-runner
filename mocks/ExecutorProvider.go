package mocks

import "gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
import "github.com/stretchr/testify/mock"

type ExecutorProvider struct {
	mock.Mock
}

func (m *ExecutorProvider) CanCreate() bool {
	ret := m.Called()

	r0 := ret.Get(0).(bool)

	return r0
}
func (m *ExecutorProvider) Create() common.Executor {
	ret := m.Called()

	r0 := ret.Get(0).(common.Executor)

	return r0
}
func (m *ExecutorProvider) Acquire(config *common.RunnerConfig) (common.ExecutorData, error) {
	ret := m.Called(config)

	r0 := ret.Get(0).(common.ExecutorData)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *ExecutorProvider) Release(config *common.RunnerConfig, data common.ExecutorData) error {
	ret := m.Called(config, data)

	r0 := ret.Error(0)

	return r0
}
func (m *ExecutorProvider) GetFeatures(features *common.FeaturesInfo) {
	m.Called(features)
}
