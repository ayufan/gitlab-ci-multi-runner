package executors

import "gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"

type DefaultExecutorProvider struct {
	common.FeaturesInfo

	Creator func() common.Executor
}

func (e DefaultExecutorProvider) CanCreate() bool {
	return e.Creator != nil
}

func (e DefaultExecutorProvider) Create() common.Executor {
	if e.Creator == nil {
		return nil
	}
	return e.Creator()
}

func (e DefaultExecutorProvider) Features() *common.FeaturesInfo {
	return &e.FeaturesInfo
}
