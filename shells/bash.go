package shells

import "github.com/ayufan/gitlab-ci-multi-runner/common"

type BashShell struct {
}

func (s *BashShell) GetName() string {
	return "bash"
}

func (s *BashShell) GenerateScript(build *common.Build) (common.ShellScript, error) {

}

func init() {
	common.RegisterShell(BashShell{})
}
