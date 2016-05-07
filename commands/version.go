package commands

import (
	"fmt"
	"runtime"
	"time"

	"github.com/codegangsta/cli"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

type VersionCommand struct {
}

func (c *VersionCommand) Execute(context *cli.Context) {
	built := time.Now()
	if common.BUILT != "now" {
		built, _ = time.Parse(time.RFC3339, common.BUILT)
	}

	fmt.Printf("Version:      %s\n", common.VERSION)
	fmt.Printf("Git revision: %s\n", common.REVISION)
	fmt.Printf("GO version:   %s\n", runtime.Version())
	fmt.Printf("Built:        %s\n", built.Format(time.RFC1123Z))
	fmt.Printf("OS/Arch:      %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

func init() {
	common.RegisterCommand2("version", "Print version details", &VersionCommand{})
}
