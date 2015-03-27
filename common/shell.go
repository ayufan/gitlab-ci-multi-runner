package common

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
)

type ShellScript struct {
	Environment []string
	Script      []byte
	Command     string
}

type Shell interface {
	GetName() string
	GenerateScript(build *Build) (*ShellScript, error)
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
