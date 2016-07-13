package virtualbox

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/ssh"
	"github.com/stretchr/testify/require"
	"os/exec"
)

func SkipIfNoVBoxManage(t *testing.T) bool {
	_, err := exec.LookPath("VBoxManage")
	if err == nil {
		return false
	}
	t.Skip("Missing VBoxManage")
	return true
}

func TestVirtualBoxExecutorRegistered(t *testing.T) {
	executors := common.GetExecutors()
	assert.Contains(t, executors, "virtualbox")
}

func TestVirtualBoxCreateExecutor(t *testing.T) {
	executor := common.NewExecutor("virtualbox")
	assert.NotNil(t, executor)
}

func TestVirtualBoxSuccessRun(t *testing.T) {
	if helpers.SkipIntegrationTest(t) {
		return
	}
	if SkipIfNoVBoxManage(t) {
		return
	}

	build := &common.Build{
		GetBuildResponse: common.GetBuildResponse{
			RepoURL:       "https://gitlab.com/gitlab-org/gitlab-test.git",
			Commands:      "echo Hello World",
			Sha:           "6907208d755b60ebeacb2e9dfea74c92c3449a1f",
			BeforeSha:     "c347ca2e140aa667b968e51ed0ffe055501fe4f4",
			RefName:       "master",
		},
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "virtualbox",
				VirtualBox: &common.VirtualBoxConfig{
					BaseName: "alpine",
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
	if helpers.SkipIntegrationTest(t) {
		return
	}
	if SkipIfNoVBoxManage(t) {
		return
	}

	build := &common.Build{
		GetBuildResponse: common.GetBuildResponse{
			RepoURL:       "https://gitlab.com/gitlab-org/gitlab-test.git",
			Commands:      "exit 1",
			Sha:           "6907208d755b60ebeacb2e9dfea74c92c3449a1f",
			BeforeSha:     "c347ca2e140aa667b968e51ed0ffe055501fe4f4",
			RefName:       "master",
		},
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "virtualbox",
				VirtualBox: &common.VirtualBoxConfig{
					BaseName: "alpine",
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
	assert.Contains(t, "Process exited with: 1", err.Error())
}

func TestVirtualBoxMissingImage(t *testing.T) {
	if helpers.SkipIntegrationTest(t) {
		return
	}
	if SkipIfNoVBoxManage(t) {
		return
	}

	build := &common.Build{
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "virtualbox",
				VirtualBox: &common.VirtualBoxConfig{
					BaseName: "non-existing-image",
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
	if helpers.SkipIntegrationTest(t) {
		return
	}
	if SkipIfNoVBoxManage(t) {
		return
	}

	build := &common.Build{
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor: "virtualbox",
				VirtualBox: &common.VirtualBoxConfig{
					BaseName: "non-existing-image",
				},
			},
		},
	}

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	require.Error(t, err)
	assert.Contains(t, "Missing SSH config", err.Error())
}
