package common

import "github.com/stretchr/testify/mock"

type MockBuildTracePatch struct {
	mock.Mock
}

func (m *MockBuildTracePatch) Patch() []byte {
	ret := m.Called()

	var r0 []byte
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]byte)
	}

	return r0
}
func (m *MockBuildTracePatch) Offset() int {
	ret := m.Called()

	r0 := ret.Get(0).(int)

	return r0
}
func (m *MockBuildTracePatch) Limit() int {
	ret := m.Called()

	r0 := ret.Get(0).(int)

	return r0
}
func (m *MockBuildTracePatch) SetNewOffset(newOffset int) {
	m.Called(newOffset)
}
