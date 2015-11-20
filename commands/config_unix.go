// +build linux darwin freebsd

package commands

import (
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"os"
	"path/filepath"
)

func getDefaultConfigDirectory() string {
	if os.Getuid() == 0 {
		return "/etc/gitlab-runner"
	} else if homeDir := helpers.GetHomeDir(); homeDir != "" {
		return filepath.Join(homeDir, ".gitlab-runner")
	} else if currentDir := helpers.GetCurrentWorkingDirectory(); currentDir != "" {
		return currentDir
	} else {
		panic("Cannot get default config file location")
	}
}
