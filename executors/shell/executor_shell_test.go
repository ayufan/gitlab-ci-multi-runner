package shell_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"os"
	"time"
)

func TestBashShellSuccessRun(t *testing.T) {
	if helpers.SkipIntegrationTests(t, "bash") {
		return
	}

	build := &common.Build{
		GetBuildResponse: common.SuccessfulBuild,
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "shell",
				Shell:    "bash",
			},
		},
	}

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	assert.NoError(t, err)
}

func TestWindowsBatchSuccessRun(t *testing.T) {
	if helpers.SkipIntegrationTests(t, "cmd.exe") {
		return
	}

	build := &common.Build{
		GetBuildResponse: common.SuccessfulBuild,
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "shell",
				Shell:    "cmd",
			},
		},
	}

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	assert.NoError(t, err)
}

func TestPowerShellSuccessRun(t *testing.T) {
	if helpers.SkipIntegrationTests(t, "powershell.exe") {
		return
	}

	build := &common.Build{
		GetBuildResponse: common.SuccessfulBuild,
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "shell",
				Shell:    "powershell",
			},
		},
	}

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	assert.NoError(t, err)
}

func TestShellBuildAbort(t *testing.T) {
	if helpers.SkipIntegrationTests(t) {
		return
	}

	build := &common.Build{
		GetBuildResponse: common.LongRunningBuild,
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "shell",
			},
		},
		SystemInterrupt: make(chan os.Signal, 1),
	}

	abortTimer := time.AfterFunc(time.Second, func() {
		t.Log("Interrupt")
		build.SystemInterrupt <- os.Interrupt
	})
	defer abortTimer.Stop()

	timeoutTimer := time.AfterFunc(time.Second*3, func() {
		t.Log("Timedout")
		t.FailNow()
	})
	defer timeoutTimer.Stop()

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	assert.EqualError(t, err, "aborted: interrupt")
}

func TestShellBuildCancel(t *testing.T) {
	if helpers.SkipIntegrationTests(t) {
		return
	}

	build := &common.Build{
		GetBuildResponse: common.LongRunningBuild,
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "shell",
			},
		},
	}

	trace := &common.Trace{Writer: os.Stdout, Abort: make(chan interface{}, 1)}

	abortTimer := time.AfterFunc(time.Second, func() {
		t.Log("Interrupt")
		trace.Abort <- true
	})
	defer abortTimer.Stop()

	timeoutTimer := time.AfterFunc(time.Second*3, func() {
		t.Log("Timedout")
		t.FailNow()
	})
	defer timeoutTimer.Stop()

	err := build.Run(&common.Config{}, trace)
	assert.EqualError(t, err, "canceled")
	assert.IsType(t, err, &common.BuildError{})
}
