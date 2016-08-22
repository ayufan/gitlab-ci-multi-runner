package common

import "github.com/stretchr/testify/mock"

import "github.com/codegangsta/cli"

type MockCommander struct {
	mock.Mock
}

func (m *MockCommander) Execute(c *cli.Context) {
	m.Called(c)
}
