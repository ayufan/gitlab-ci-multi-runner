package vbox

import (
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"github.com/stretchr/testify/assert"
	"testing"
)


func TestVboxExecutorRegistered(t *testing.T) {
	executors := common.GetExecutors()
	assert.Contains(t, executors, "vbox")
}

func TestVboxCreateExecutor(t *testing.T) {
	executor := common.NewExecutor("vbox")
	assert.NotNil(t, executor)
}
