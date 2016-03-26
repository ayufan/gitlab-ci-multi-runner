package common

import (
	log "github.com/Sirupsen/logrus"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
)

type ShellScript struct {
	Environment []string
	Command     string
	Arguments   []string
	Script      string
	Extension   string
}

type ShellType int

const (
	NormalShell ShellType = iota
	LoginShell
)

func (s *ShellScript) GetCommandWithArguments() []string {
	parts := []string{s.Command}
	for _, arg := range s.Arguments {
		parts = append(parts, arg)
	}
	return parts
}

func (s *ShellScript) String() string {
	return helpers.ToYAML(s)
}

type Shell interface {
	GetName() string
	GetSupportedOptions() []string
	GetFeatures(features *FeaturesInfo)
	IsDefault() bool

	PreBuild(build *Build, options BuildOptions) (*ShellScript, error)
	Build(build *Build, options BuildOptions) (*ShellScript, error)
	PostBuild(build *Build, options BuildOptions) (*ShellScript, error)
}

var shells map[string]Shell

func RegisterShell(shell Shell) {
	log.Debugln("Registering", shell.GetName(), "shell...")

	if shells == nil {
		shells = make(map[string]Shell)
	}
	if shells[shell.GetName()] != nil {
		panic("Shell already exist: " + shell.GetName())
	}
	shells[shell.GetName()] = shell
}

func GetShell(shell string) Shell {
	if shells == nil {
		return nil
	}

	return shells[shell]
}

func GetShells() []string {
	names := []string{}
	if shells != nil {
		for name := range shells {
			names = append(names, name)
		}
	}
	return names
}

func GetDefaultShell() string {
	if shells == nil {
		panic("no shells defined")
	}

	for _, shell := range shells {
		if shell.IsDefault() {
			return shell.GetName()
		}
	}
	panic("no default shell defined")
}
