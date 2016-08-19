package common

import "github.com/stretchr/testify/mock"

type MockBuildTrace struct {
	mock.Mock
}

func (m *MockBuildTrace) Success() {
	m.Called()
}
func (m *MockBuildTrace) Fail(err error) {
	m.Called(err)
}
func (m *MockBuildTrace) Aborted() chan interface{} {
	ret := m.Called()

	var r0 chan interface{}
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(chan interface{})
	}

	return r0
}
func (m *MockBuildTrace) IsStdout() bool {
	ret := m.Called()

	r0 := ret.Get(0).(bool)

	return r0
}
