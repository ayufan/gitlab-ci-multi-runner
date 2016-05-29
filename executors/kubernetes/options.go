package kubernetes

import (
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

type Options interface {
	Privileged() bool
}

type DefaultOptions struct {
	common.BuildVariables
}

func (o DefaultOptions) Privileged() bool {
	return o.Get("privileged") == "true"
}
