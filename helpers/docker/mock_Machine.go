package docker_helpers

import "github.com/stretchr/testify/mock"

type MockMachine struct {
	mock.Mock
}

func (m *MockMachine) Create(driver string, name string, opts ...string) error {
	ret := m.Called(driver, name, opts)

	r0 := ret.Error(0)

	return r0
}
func (m *MockMachine) Provision(name string) error {
	ret := m.Called(name)

	r0 := ret.Error(0)

	return r0
}
func (m *MockMachine) Remove(name string) error {
	ret := m.Called(name)

	r0 := ret.Error(0)

	return r0
}
func (m *MockMachine) List(nodeFilter string) ([]string, error) {
	ret := m.Called(nodeFilter)

	var r0 []string
	if ret.Get(0) != nil {
		r0 = ret.Get(0).([]string)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *MockMachine) Exist(name string) bool {
	ret := m.Called(name)

	r0 := ret.Get(0).(bool)

	return r0
}
func (m *MockMachine) CanConnect(name string) bool {
	ret := m.Called(name)

	r0 := ret.Get(0).(bool)

	return r0
}
func (m *MockMachine) Credentials(name string) (DockerCredentials, error) {
	ret := m.Called(name)

	r0 := ret.Get(0).(DockerCredentials)
	r1 := ret.Error(1)

	return r0, r1
}
