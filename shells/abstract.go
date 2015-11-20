package shells

import (
	. "gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

type AbstractShell struct {
	SupportedOptions []string
}

func (s *AbstractShell) GetSupportedOptions() []string {
	return s.SupportedOptions
}

func (s *AbstractShell) GetVariables(info ShellScriptInfo) []string {
	return info.Build.GetAllVariables().StringList()
}

func (s *AbstractShell) GetFeatures(features *FeaturesInfo) {
}
