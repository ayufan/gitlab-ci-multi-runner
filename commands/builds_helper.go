package commands

import (
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"sync"
)

type buildsHelper struct {
	builds     []*common.Build
	buildsLock sync.Mutex
}

func (b *buildsHelper) count(runner *common.RunnerConfig) int {
	count := 0
	for _, build := range b.builds {
		if build.Runner.ShortDescription() == runner.ShortDescription() {
			count++
		}
	}
	return count
}

func (b *buildsHelper) acquire(runner *runnerAcquire) (build *common.Build) {
	b.buildsLock.Lock()
	defer b.buildsLock.Unlock()

	// Check number of builds
	count := b.count(&runner.RunnerConfig)
	if runner.Limit > 0 && count >= runner.Limit {
		// Too many builds
		return
	}

	// Create a new build
	build = &common.Build{
		Runner:       &runner.RunnerConfig,
		ExecutorData: runner.data,
	}
	build.AssignID(b.builds...)
	b.builds = append(b.builds, build)
	return
}

func (b *buildsHelper) release(deleteBuild *common.Build) bool {
	b.buildsLock.Lock()
	defer b.buildsLock.Unlock()

	for idx, build := range b.builds {
		if build == deleteBuild {
			b.builds = append(b.builds[0:idx], b.builds[idx+1:]...)
			return true
		}
	}
	return false
}
