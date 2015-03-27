package shells

import (
	"errors"
	"github.com/ayufan/gitlab-ci-multi-runner/common"
)

type CmdShell struct {
}

func (c *CmdShell) GetName() string {
	return "cmd"
}

func (c *CmdShell) GenerateScript(build *common.Build) (*common.ShellScript, error) {
	return nil, errors.New("not yet supported")
}

func init() {
	common.RegisterShell(&CmdShell{})
}
