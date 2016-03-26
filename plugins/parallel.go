package plugins

import "gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"

type parallelPlugin struct {
}

func (s *parallelPlugin) GetName() string {
	return "clone"
}

func (s *parallelPlugin) Run(b *common.Build, options common.BuildOptions, abort chan error) error {
	steps, ok := options.Get("steps")
}

func init() {
	common.RegisterPlugin(&parallelPlugin{})
}
