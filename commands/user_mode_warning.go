package commands

import (
	"os"
	"runtime"

	"github.com/Sirupsen/logrus"
)

func userModeWarning(withRun bool) {
	logrus.WithFields(logrus.Fields{
		"GOOS": runtime.GOOS,
		"uid":  os.Getuid(),
	}).Debugln("Checking runtime mode")

	// everything is supported on windows
	if runtime.GOOS == "windows" {
		return
	}

	systemMode := os.Getuid() == 0

	// We support services on Linux, Windows and Darwin
	noServices :=
		runtime.GOOS != "linux" &&
			runtime.GOOS != "darwin"

	// We don't support services installed as an User on Linux
	noUserService :=
		!systemMode &&
			runtime.GOOS == "linux"

	if systemMode {
		logrus.Infoln("Running in system-mode.")
	} else {
		logrus.Warningln("Running in user-mode.")
	}

	if withRun {
		if noServices {
			logrus.Warningln("You need to manually start builds processing:")
			logrus.Warningln("$ gitlab-runner run")
		} else if noUserService {
			logrus.Warningln("The user-mode requires you to manually start builds processing:")
			logrus.Warningln("$ gitlab-runner run")
		}
	}

	if !systemMode {
		logrus.Warningln("Use sudo for system-mode:")
		logrus.Warningln("$ sudo gitlab-runner...")
	}
	logrus.Infoln("")
}
