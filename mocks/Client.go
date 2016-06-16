package mocks

import "github.com/stretchr/testify/mock"

import "github.com/fsouza/go-dockerclient"

type Client struct {
	mock.Mock
}

func (m *Client) InspectImage(name string) (*docker.Image, error) {
	ret := m.Called(name)

	var r0 *docker.Image
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*docker.Image)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Client) PullImage(opts docker.PullImageOptions, auth docker.AuthConfiguration) error {
	ret := m.Called(opts, auth)

	r0 := ret.Error(0)

	return r0
}
func (m *Client) ImportImage(opts docker.ImportImageOptions) error {
	ret := m.Called(opts)

	r0 := ret.Error(0)

	return r0
}
func (m *Client) CreateContainer(opts docker.CreateContainerOptions) (*docker.Container, error) {
	ret := m.Called(opts)

	var r0 *docker.Container
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*docker.Container)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Client) StartContainer(id string, hostConfig *docker.HostConfig) error {
	ret := m.Called(id, hostConfig)

	r0 := ret.Error(0)

	return r0
}
func (m *Client) WaitContainer(id string) (int, error) {
	ret := m.Called(id)

	r0 := ret.Get(0).(int)
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Client) KillContainer(opts docker.KillContainerOptions) error {
	ret := m.Called(opts)

	r0 := ret.Error(0)

	return r0
}
func (m *Client) InspectContainer(id string) (*docker.Container, error) {
	ret := m.Called(id)

	var r0 *docker.Container
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*docker.Container)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Client) AttachToContainer(opts docker.AttachToContainerOptions) error {
	ret := m.Called(opts)

	r0 := ret.Error(0)

	return r0
}
func (m *Client) RemoveContainer(opts docker.RemoveContainerOptions) error {
	ret := m.Called(opts)

	r0 := ret.Error(0)

	return r0
}
func (m *Client) Logs(opts docker.LogsOptions) error {
	ret := m.Called(opts)

	r0 := ret.Error(0)

	return r0
}
func (m *Client) Info() (*docker.Env, error) {
	ret := m.Called()

	var r0 *docker.Env
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*docker.Env)
	}
	r1 := ret.Error(1)

	return r0, r1
}
