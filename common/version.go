package common

import (
	"fmt"
	"runtime"
	"time"

	"github.com/codegangsta/cli"
)

var NAME = "gitlab-ci-multi-runner"
var VERSION = "dev"
var REVISION = "HEAD"
var BUILT = "now"
var BRANCH = "HEAD"

func VersionPrinter(c *cli.Context) {
	fmt.Print(ExtendedVersion())
}

func VersionLine() string {
	return fmt.Sprintf("%s %s (%s)", NAME, VERSION, REVISION)
}

func VersionShortLine() string {
	return fmt.Sprintf("%s (%s)", VERSION, REVISION)
}

func VersionUserAgent() string {
	return fmt.Sprintf("%s %s (%s; %s; %s/%s)", NAME, VERSION, BRANCH, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

func ExtendedVersion() string {
	built := time.Now()
	if BUILT != "now" {
		built, _ = time.Parse(time.RFC3339, BUILT)
	}

	version := fmt.Sprintf("Version:      %s\n", VERSION)
	version += fmt.Sprintf("Git revision: %s\n", REVISION)
	version += fmt.Sprintf("Git branch:   %s\n", BRANCH)
	version += fmt.Sprintf("GO version:   %s\n", runtime.Version())
	version += fmt.Sprintf("Built:        %s\n", built.Format(time.RFC1123Z))
	version += fmt.Sprintf("OS/Arch:      %s/%s\n", runtime.GOOS, runtime.GOARCH)

	return version
}
