package machine

import (
	"crypto/rand"
	"fmt"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"time"
)

func machineFormat(runner string, template string) string {
	if runner != "" {
		return "runner-" + runner + "-" + template
	}
	return template
}

func machineFilter(config *common.RunnerConfig) string {
	return machineFormat(config.ShortDescription(), config.Machine.MachineName)
}

func newMachineName(machineFilter string) string {
	r := make([]byte, 4)
	rand.Read(r)
	t := time.Now().Unix()
	return fmt.Sprintf(machineFilter, fmt.Sprintf("%d-%x", t, r))
}
