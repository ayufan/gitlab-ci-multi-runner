package plugins

import "gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"

type clonePlugin struct {
}

func (s *clonePlugin) GetName() string {
	return "clone"
}

func (s *clonePlugin) Run(b *common.Build, abort chan error) error {
	sc, err := b.Shell.PreBuild(b)
	if err != nil {
		return err
	}
	return b.Step(sc, common.ImagePreBuild, abort)
}

func init() {
	common.RegisterPlugin(&clonePlugin{})
}
