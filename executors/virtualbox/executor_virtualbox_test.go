package virtualbox

import (
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"github.com/stretchr/testify/assert"
	"testing"
)


func TestVirtualBoxExecutorRegistered(t *testing.T) {
	executors := common.GetExecutors()
	assert.Contains(t, executors, "virtualbox")
}

func TestVirtualBoxCreateExecutor(t *testing.T) {
	executor := common.NewExecutor("virtualbox")
	assert.NotNil(t, executor)
}
