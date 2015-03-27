package common

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/ayufan/gitlab-ci-multi-runner/helpers"
	"strings"
)

type ShellScript struct {
	Environment []string
	Script      []byte
	Command     string
	Arguments   []string
	PassFile    bool
	Extension   string
}

func (s *ShellScript) GetCommandWithArguments() []string {
	parts := []string{s.Command}
	for _, arg := range s.Arguments {
		parts = append(parts, helpers.ShellEscape(arg))
	}
	return parts
}

func (s *ShellScript) GetFullCommand() string {
	parts := s.GetCommandWithArguments()
	for idx, part := range parts {
		parts[idx] = helpers.ShellEscape(part)
	}
	return strings.Join(parts, " ")
}

type Shell interface {
	GetName() string
	GenerateScript(build *Build) (*ShellScript, error)
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

func GenerateShellScript(name string, build *Build) (*ShellScript, error) {
	shell := GetShell(name)
	if shell == nil {
		return nil, fmt.Errorf("shell %s not found", name)
	}

	return shell.GenerateScript(build)
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
