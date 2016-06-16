package mocks

import "gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
import "github.com/stretchr/testify/mock"
import "io"

type Network struct {
	mock.Mock
}

func (m *Network) GetBuild(config common.RunnerConfig) (*common.GetBuildResponse, bool) {
	ret := m.Called(config)

	var r0 *common.GetBuildResponse
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.GetBuildResponse)
	}
	r1 := ret.Get(1).(bool)

	return r0, r1
}
func (m *Network) RegisterRunner(config common.RunnerCredentials, description string, tags string) *common.RegisterRunnerResponse {
	ret := m.Called(config, description, tags)

	var r0 *common.RegisterRunnerResponse
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*common.RegisterRunnerResponse)
	}

	return r0
}
func (m *Network) DeleteRunner(config common.RunnerCredentials) bool {
	ret := m.Called(config)

	r0 := ret.Get(0).(bool)

	return r0
}
func (m *Network) VerifyRunner(config common.RunnerCredentials) bool {
	ret := m.Called(config)

	r0 := ret.Get(0).(bool)

	return r0
}
func (m *Network) UpdateBuild(config common.RunnerConfig, id int, state common.BuildState, trace *string) common.UpdateState {
	ret := m.Called(config, id, state, trace)

	r0 := ret.Get(0).(common.UpdateState)

	return r0
}
func (m *Network) PatchTrace(config common.RunnerConfig, buildCredentials *common.BuildCredentials, tracePatch common.BuildTracePatch) common.UpdateState {
	ret := m.Called(config, buildCredentials, tracePatch)

	r0 := ret.Get(0).(common.UpdateState)

	return r0
}
func (m *Network) DownloadArtifacts(config common.BuildCredentials, artifactsFile string) common.DownloadState {
	ret := m.Called(config, artifactsFile)

	r0 := ret.Get(0).(common.DownloadState)

	return r0
}
func (m *Network) UploadRawArtifacts(config common.BuildCredentials, reader io.Reader, baseName string, expireIn string) common.UploadState {
	ret := m.Called(config, reader, baseName, expireIn)

	r0 := ret.Get(0).(common.UploadState)

	return r0
}
func (m *Network) UploadArtifacts(config common.BuildCredentials, artifactsFile string) common.UploadState {
	ret := m.Called(config, artifactsFile)

	r0 := ret.Get(0).(common.UploadState)

	return r0
}
func (m *Network) ProcessBuild(config common.RunnerConfig, buildCredentials *common.BuildCredentials) common.BuildTrace {
	ret := m.Called(config, buildCredentials)

	r0 := ret.Get(0).(common.BuildTrace)

	return r0
}
