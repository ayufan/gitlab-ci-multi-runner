package parallels_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/ssh"
)

const prlImage = "ubuntu-runner"
const prlCtl = "prlctl"

var prlSshConfig = &ssh.Config{
	User:     "vagrant",
	Password: "vagrant",
}

func TestParallelsExecutorRegistered(t *testing.T) {
	executors := common.GetExecutors()
	assert.Contains(t, executors, "parallels")
}

func TestParallelsCreateExecutor(t *testing.T) {
	executor := common.NewExecutor("parallels")
	assert.NotNil(t, executor)
}

func TestParallelsSuccessRun(t *testing.T) {
	if helpers.SkipIntegrationTests(t, prlCtl, "--version") {
		return
	}

	build := &common.Build{
		GetBuildResponse: common.SuccessfulBuild,
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "parallels",
				Parallels: &common.ParallelsConfig{
					BaseName:         prlImage,
					DisableSnapshots: true,
				},
				SSH: prlSshConfig,
			},
		},
	}

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	assert.NoError(t, err, "Make sure that you have done 'make -C tests/ubuntu parallels'")
}

func TestParallelsBuildFail(t *testing.T) {
	if helpers.SkipIntegrationTests(t, prlCtl, "--version") {
		return
	}

	build := &common.Build{
		GetBuildResponse: common.FailedBuild,
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "parallels",
				Parallels: &common.ParallelsConfig{
					BaseName:         prlImage,
					DisableSnapshots: true,
				},
				SSH: prlSshConfig,
			},
		},
	}

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	require.Error(t, err, "error")
	assert.IsType(t, err, &common.BuildError{})
	assert.Contains(t, err.Error(), "Process exited with: 1")
}

func TestParallelsMissingImage(t *testing.T) {
	if helpers.SkipIntegrationTests(t, prlCtl, "--version") {
		return
	}

	build := &common.Build{
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "parallels",
				Parallels: &common.ParallelsConfig{
					BaseName:         "non-existing-image",
					DisableSnapshots: true,
				},
				SSH: prlSshConfig,
			},
		},
	}

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Could not find a registered machine named")
}

func TestParallelsMissingSSHCredentials(t *testing.T) {
	if helpers.SkipIntegrationTests(t, prlCtl, "--version") {
		return
	}

	build := &common.Build{
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "parallels",
				Parallels: &common.ParallelsConfig{
					BaseName:         "non-existing-image",
					DisableSnapshots: true,
				},
			},
		},
	}

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Missing SSH config")
}

func TestParallelsBuildAbort(t *testing.T) {
	if helpers.SkipIntegrationTests(t, prlCtl, "--version") {
		return
	}

	build := &common.Build{
		GetBuildResponse: common.LongRunningBuild,
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "parallels",
				Parallels: &common.ParallelsConfig{
					BaseName:         prlImage,
					DisableSnapshots: true,
				},
				SSH: prlSshConfig,
			},
		},
		SystemInterrupt: make(chan os.Signal, 1),
	}

	abortTimer := time.AfterFunc(time.Second, func() {
		t.Log("Interrupt")
		build.SystemInterrupt <- os.Interrupt
	})
	defer abortTimer.Stop()

	timeoutTimer := time.AfterFunc(time.Minute, func() {
		t.Log("Timedout")
		t.FailNow()
	})
	defer timeoutTimer.Stop()

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	assert.EqualError(t, err, "aborted: interrupt")
}

func TestParallelsBuildCancel(t *testing.T) {
	if helpers.SkipIntegrationTests(t, prlCtl, "--version") {
		return
	}

	build := &common.Build{
		GetBuildResponse: common.LongRunningBuild,
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "parallels",
				Parallels: &common.ParallelsConfig{
					BaseName:         prlImage,
					DisableSnapshots: true,
				},
				SSH: prlSshConfig,
			},
		},
	}

	trace := &common.Trace{Writer: os.Stdout, Abort: make(chan interface{}, 1)}

	abortTimer := time.AfterFunc(time.Second, func() {
		t.Log("Interrupt")
		trace.Abort <- true
	})
	defer abortTimer.Stop()

	timeoutTimer := time.AfterFunc(time.Minute, func() {
		t.Log("Timedout")
		t.FailNow()
	})
	defer timeoutTimer.Stop()

	err := build.Run(&common.Config{}, trace)
	assert.IsType(t, err, &common.BuildError{})
	assert.EqualError(t, err, "canceled")
}
