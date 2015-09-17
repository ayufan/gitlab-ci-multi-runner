package mocks

import "github.com/ayufan/golang-kardianos-service"
import "github.com/stretchr/testify/mock"

type Interface struct {
	mock.Mock
}

func (m *Interface) Start(s service.Service) error {
	ret := m.Called(s)

	r0 := ret.Error(0)

	return r0
}
func (m *Interface) Stop(s service.Service) error {
	ret := m.Called(s)

	r0 := ret.Error(0)

	return r0
}
