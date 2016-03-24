package plugins

import "gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"

type defaultPlugin struct {
}

func (s *defaultPlugin) GetName() string {
	return "default"
}

func (s *defaultPlugin) Run(b *common.Build, abort chan error) (err error) {
	clone := &clonePlugin{}
	err = clone.Run(b, abort)
	if err != nil {
		return
	}

	script := &scriptPlugin{}
	err = script.Run(b, abort)
	if err != nil {
		return
	}

	artifacts := &artifactsPlugin{}
	err = artifacts.Run(b, abort)
	if err != nil {
		return
	}
	return
}

func init() {
	common.RegisterPlugin(&defaultPlugin{})
}
