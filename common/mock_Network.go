package common

import "github.com/stretchr/testify/mock"

import "io"

type MockNetwork struct {
	mock.Mock
}

func (m *MockNetwork) GetBuild(config RunnerConfig) (*GetBuildResponse, bool) {
	ret := m.Called(config)

	var r0 *GetBuildResponse
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*GetBuildResponse)
	}
	r1 := ret.Get(1).(bool)

	return r0, r1
}
func (m *MockNetwork) RegisterRunner(config RunnerCredentials, description string, tags string) *RegisterRunnerResponse {
	ret := m.Called(config, description, tags)

	var r0 *RegisterRunnerResponse
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*RegisterRunnerResponse)
	}

	return r0
}
func (m *MockNetwork) DeleteRunner(config RunnerCredentials) bool {
	ret := m.Called(config)

	r0 := ret.Get(0).(bool)

	return r0
}
func (m *MockNetwork) VerifyRunner(config RunnerCredentials) bool {
	ret := m.Called(config)

	r0 := ret.Get(0).(bool)

	return r0
}
func (m *MockNetwork) UpdateBuild(config RunnerConfig, id int, state BuildState, trace *string) UpdateState {
	ret := m.Called(config, id, state, trace)

	r0 := ret.Get(0).(UpdateState)

	return r0
}
func (m *MockNetwork) PatchTrace(config RunnerConfig, buildCredentials *BuildCredentials, tracePart BuildTracePatch) UpdateState {
	ret := m.Called(config, buildCredentials, tracePart)

	r0 := ret.Get(0).(UpdateState)

	return r0
}
func (m *MockNetwork) DownloadArtifacts(config BuildCredentials, artifactsFile string) DownloadState {
	ret := m.Called(config, artifactsFile)

	r0 := ret.Get(0).(DownloadState)

	return r0
}
func (m *MockNetwork) UploadRawArtifacts(config BuildCredentials, reader io.Reader, baseName string, expireIn string) UploadState {
	ret := m.Called(config, reader, baseName, expireIn)

	r0 := ret.Get(0).(UploadState)

	return r0
}
func (m *MockNetwork) UploadArtifacts(config BuildCredentials, artifactsFile string) UploadState {
	ret := m.Called(config, artifactsFile)

	r0 := ret.Get(0).(UploadState)

	return r0
}
func (m *MockNetwork) ProcessBuild(config RunnerConfig, buildCredentials *BuildCredentials) BuildTrace {
	ret := m.Called(config, buildCredentials)

	r0 := ret.Get(0).(BuildTrace)

	return r0
}
