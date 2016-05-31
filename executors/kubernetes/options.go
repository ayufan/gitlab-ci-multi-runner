package kubernetes

import (
	"strings"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

type Options interface {
	Privileged() bool
}

type DefaultOptions struct {
	common.BuildVariables
}

func (o DefaultOptions) Privileged() bool {
	return strings.ToLower(o.Get("privileged")) == "true"
}
