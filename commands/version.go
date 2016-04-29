package commands

import (
	"fmt"
	"runtime"

	"github.com/codegangsta/cli"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

type VersionCommand struct {
}

func (c *VersionCommand) Execute(context *cli.Context) {
	fmt.Printf("Version:      %s\n", common.VERSION)
	fmt.Printf("Git revision: %s\n", common.REVISION)
	fmt.Printf("GO version:   %s\n", runtime.Version())
	fmt.Printf("Built:        %s\n", common.BUILT)
	fmt.Printf("OS/Arch:      %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

func init() {
	common.RegisterCommand2("version", "Print version details", &VersionCommand{})
}
