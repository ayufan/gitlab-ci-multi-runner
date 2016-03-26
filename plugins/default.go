package plugins

import (
	"errors"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

type defaultPlugin struct {
}

func (s *defaultPlugin) GetName() string {
	return "default"
}

func (s *defaultPlugin) Run(b *common.Build, options common.BuildOptions, abort chan error) (err error) {
	preBuild, err := b.Shell.PreBuild(b)
	if err != nil {
		return
	}
	err = b.Step(preBuild, common.ImagePreBuild, abort)
	if err != nil {
		return
	}

	plugin := common.GetPlugin(b.Plugin)
	if plugin == nil {
		return errors.New("plugin not found: " + b.Plugin)
	}
	err = plugin.Run(b, options, abort)
	if err != nil {
		return
	}

	postBuild, err := b.Shell.PreBuild(b)
	if err != nil {
		return
	}
	err = b.Step(postBuild, common.ImagePostBuild, abort)
	if err != nil {
		return
	}
	return
}

func init() {
	common.RegisterPlugin(&defaultPlugin{})
}
