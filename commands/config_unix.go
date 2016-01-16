// +build linux darwin freebsd

package commands

import (
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"os"
	"path/filepath"
	"github.com/Sirupsen/logrus"
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

func userModeWarning(withRun bool) {
	if os.Getuid() == 0 {
		logrus.Infoln("Running in system-mode.")
	} else {
		logrus.Warningln("Running in user-mode.")
		if withRun {
			logrus.Warningln("The user-mode requires you to manually start builds processing:")
			logrus.Warningln("$ gitlab-runner run")
		}
		logrus.Warningln("Use sudo for system-mode:")
		logrus.Warningln("$ sudo gitlab-runner...")
	}
}
