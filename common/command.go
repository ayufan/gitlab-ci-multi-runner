package common

import (
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"gitlab.com/ayufan/golang-cli-helpers"
)

var commands []cli.Command

type Commander interface {
	Execute(c *cli.Context)
}

func RegisterCommand(command cli.Command) {
	log.Debugln("Registering", command.Name, "command...")
	commands = append(commands, command)
}

func RegisterCommand2(name, usage string, data Commander, flags ...cli.Flag) {
	RegisterCommand(cli.Command{
		Name:   name,
		Usage:  usage,
		Action: data.Execute,
		Flags:  append(flags, clihelpers.GetFlagsFromStruct(data)...),
	})
}

func GetCommands() []cli.Command {
	return commands
}
