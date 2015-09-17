// +build linux darwin freebsd

package commands

import (
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"os"
	"path/filepath"
)

func getDefaultConfigFile() string {
	if os.Getuid() == 0 {
		return "/etc/gitlab-runner/config.toml"
	} else if homeDir := helpers.GetHomeDir(); homeDir != "" {
		return filepath.Join(homeDir, ".gitlab-runner", "config.toml")
	} else if currentDir := helpers.GetCurrentWorkingDirectory(); currentDir != "" {
		return filepath.Join(currentDir, "config.toml")
	} else {
		panic("Cannot get default config file location")
	}
}
