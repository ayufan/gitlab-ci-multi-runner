package common

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"strings"
)

type ShellScript struct {
	Environment []string
	Script      string
	Command     string
	Arguments   []string
	PassFile    bool
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

func (s *ShellScript) GetFullCommand() string {
	parts := s.GetCommandWithArguments()
	for idx, part := range parts {
		parts[idx] = part
	}
	return strings.Join(parts, " ")
}

func (s *ShellScript) GetScriptBytes() []byte {
	return []byte(s.Script)
}

func (s *ShellScript) String() string {
	return helpers.ToYAML(s)
}

type ShellScriptInfo struct {
	Shell       string
	Build       *Build
	Type        ShellType
	User        *string
	Environment []BuildVariable
}

type Shell interface {
	GetName() string
	GetSupportedOptions() []string
	GenerateScript(info ShellScriptInfo) (*ShellScript, error)
	GetFeatures(features *FeaturesInfo)
	IsDefault() bool
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

func GenerateShellScript(info ShellScriptInfo) (*ShellScript, error) {
	shell := GetShell(info.Shell)
	if shell == nil {
		return nil, fmt.Errorf("shell %s not found", info.Shell)
	}

	return shell.GenerateScript(info)
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
