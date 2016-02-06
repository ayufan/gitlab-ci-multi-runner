package executors

import (
	// make sure that shells get loaded before executors
	// this happens, because of difference in ordering init()
	// from external packages between 1.4.x and 1.5.x
	// this import forces to load shells before
	// and fixes: panic: no shells defined
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/shells"
)
