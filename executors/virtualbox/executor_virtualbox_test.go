package virtualbox_test

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

const vboxManage = "vboxmanage"

func TestVirtualBoxExecutorRegistered(t *testing.T) {
	executors := common.GetExecutors()
	assert.Contains(t, executors, "virtualbox")
}

func TestVirtualBoxCreateExecutor(t *testing.T) {
	executor := common.NewExecutor("virtualbox")
	assert.NotNil(t, executor)
}

func TestVirtualBoxSuccessRun(t *testing.T) {
	if helpers.SkipIntegrationTests(t, vboxManage, "--version") {
		return
	}

	build := &common.Build{
		GetBuildResponse: common.SuccessfulBuild,
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "virtualbox",
				VirtualBox: &common.VirtualBoxConfig{
					BaseName:         "alpine",
					DisableSnapshots: true,
				},
				SSH: &ssh.Config{
					User:     "root",
					Password: "root",
				},
			},
		},
	}

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	assert.NoError(t, err, "Make sure that you have alpine.ova installed in your VirtualBox")
}

func TestVirtualBoxBuildFail(t *testing.T) {
	if helpers.SkipIntegrationTests(t, vboxManage, "--version") {
		return
	}

	build := &common.Build{
		GetBuildResponse: common.FailedBuild,
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "virtualbox",
				VirtualBox: &common.VirtualBoxConfig{
					BaseName:         "alpine",
					DisableSnapshots: true,
				},
				SSH: &ssh.Config{
					User:     "root",
					Password: "root",
				},
			},
		},
	}

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	require.Error(t, err, "error")
	assert.IsType(t, err, &common.BuildError{})
	assert.Contains(t, "Process exited with: 1", err.Error())
}

func TestVirtualBoxMissingImage(t *testing.T) {
	if helpers.SkipIntegrationTests(t, vboxManage, "--version") {
		return
	}

	build := &common.Build{
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "virtualbox",
				VirtualBox: &common.VirtualBoxConfig{
					BaseName:         "non-existing-image",
					DisableSnapshots: true,
				},
				SSH: &ssh.Config{
					User:     "root",
					Password: "root",
				},
			},
		},
	}

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	require.Error(t, err)
	assert.Contains(t, "Could not find a registered machine named", err.Error())
}

func TestVirtualBoxMissingSSHCredentials(t *testing.T) {
	if helpers.SkipIntegrationTests(t, vboxManage, "--version") {
		return
	}

	build := &common.Build{
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "virtualbox",
				VirtualBox: &common.VirtualBoxConfig{
					BaseName:         "non-existing-image",
					DisableSnapshots: true,
				},
			},
		},
	}

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	require.Error(t, err)
	assert.Contains(t, "Missing SSH config", err.Error())
}

func TestVirtualBoxBuildAbort(t *testing.T) {
	if helpers.SkipIntegrationTests(t, vboxManage, "--version") {
		return
	}

	build := &common.Build{
		GetBuildResponse: common.LongRunningBuild,
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "virtualbox",
				VirtualBox: &common.VirtualBoxConfig{
					BaseName:         "alpine",
					DisableSnapshots: true,
				},
				SSH: &ssh.Config{
					User:     "root",
					Password: "root",
				},
			},
		},
		BuildAbort: make(chan os.Signal, 1),
	}

	abortTimer := time.AfterFunc(time.Second, func() {
		t.Log("Interrupt")
		build.BuildAbort <- os.Interrupt
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

func TestVirtualBoxBuildCancel(t *testing.T) {
	if helpers.SkipIntegrationTests(t, vboxManage, "--version") {
		return
	}

	build := &common.Build{
		GetBuildResponse: common.LongRunningBuild,
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "virtualbox",
				VirtualBox: &common.VirtualBoxConfig{
					BaseName:         "alpine",
					DisableSnapshots: true,
				},
				SSH: &ssh.Config{
					User:     "root",
					Password: "root",
				},
			},
		},
	}

	trace := &common.Trace{Writer: os.Stdout}

	abortTimer := time.AfterFunc(time.Second, func() {
		t.Log("Interrupt")
		for trace.Abort == nil {
			time.Sleep(time.Second)
		}
		trace.Abort()
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
