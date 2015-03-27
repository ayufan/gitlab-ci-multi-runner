package common

import (
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

var commands []cli.Command

func RegisterCommand(command cli.Command) {
	log.Debugln("Registering", command.Name, "command...")
	commands = append(commands, command)
}

func GetCommands() []cli.Command {
	return commands
}
