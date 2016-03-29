package shells

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/mocks"
)

func TestWriteTLSCAInfo(t *testing.T) {
	w := &mocks.ShellWriter{}
	defer w.AssertExpectations(t)

	b := &AbstractShell{}

	build := &common.Build{
		GetBuildResponse: common.GetBuildResponse{
			TLSCAChain: "chain",
		},
		Runner: &common.RunnerConfig{
			RunnerCredentials: common.RunnerCredentials{
				URL: "https://example.com:440/ci",
			},
		},
	}

	expectVariable := func(key string) {
		w.On("Variable", common.BuildVariable{
			Key:      key,
			Value:    "chain",
			Public:   true,
			Internal: true,
			File:     true,
		}).Return().Once()
	}

	expectVariable("KEY")
	err := b.writeTLSCAInfo(w, build, "KEY", "https://example.com:440/ci")
	assert.NoError(t, err)

	expectVariable("KEY2")
	err = b.writeTLSCAInfo(w, build, "KEY2", "https://EXAMPLE.com:440/ci")
	assert.NoError(t, err)

	expectVariable("KEY8")
	err = b.writeTLSCAInfo(w, build, "KEY8", "https://user:password@EXAMPLE.com:440/ci")
	assert.NoError(t, err)

	err = b.writeTLSCAInfo(w, build, "KEY3", "http://EXAMPLE.com:440/ci")
	assert.NoError(t, err)

	err = b.writeTLSCAInfo(w, build, "KEY4", "http://EXAMPLE.com/ci")
	assert.NoError(t, err)

	err = b.writeTLSCAInfo(w, build, "KEY5", "https://SERVER.com/")
	assert.NoError(t, err)

	err = b.writeTLSCAInfo(w, build, "KEY6", "https://SERVER.com/")
	assert.NoError(t, err)

	err = b.writeTLSCAInfo(w, build, "KEY7", "https://other-ServeR.com/")
	assert.NoError(t, err)
}
