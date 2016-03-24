package plugins

import "gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"

type artifactsPlugin struct {
}

func (s *artifactsPlugin) GetName() string {
	return "artifacts"
}

func (s *artifactsPlugin) Run(b *common.Build, abort chan error) error {
	sc, err := b.Shell.PostBuild(b)
	if err != nil {
		return err
	}
	return b.Step(sc, common.ImagePostBuild, abort)
}

func init() {
	common.RegisterPlugin(&artifactsPlugin{})
}
