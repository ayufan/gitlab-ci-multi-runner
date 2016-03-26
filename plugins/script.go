package plugins

import "gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"

type scriptPlugin struct {
}

func (s *scriptPlugin) GetName() string {
	return "script"
}

func (s *scriptPlugin) Run(b *common.Build, options common.BuildOptions, abort chan error) error {
	sc, err := b.Shell.Build(b)
	if err != nil {
		return err
	}
	return b.Step(sc, common.ImageDefault, abort)
}

func init() {
	common.RegisterPlugin(&scriptPlugin{})
}
