package kubernetes

import (
	"strings"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

// Options is an interface that can be implemented to provide additional
// config options during a build
type Options interface {
	// Privileged returns whether this build should run in privileged mode
	Privileged() bool
}

// DefaultOptions is the default implementation of the Options interface,
// which reads it's options from a list of BuildVariables
type DefaultOptions struct {
	common.BuildVariables
}

// Privileged returns whether this build should run in privileged mode.
// It ignores case and tests if the 'privileged' env var is 'true'
func (o DefaultOptions) Privileged() bool {
	return strings.ToLower(o.Get("privileged")) == "true"
}
