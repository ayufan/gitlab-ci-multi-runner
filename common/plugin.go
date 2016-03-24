package common

import "github.com/Sirupsen/logrus"

type Plugin interface {
	GetName() string
	Run(b *Build, abort chan error) error
}

var plugins map[string]Plugin

func RegisterPlugin(plugin Plugin) {
	logrus.Debugln("Registering", plugin.GetName(), "plugin...")

	if plugins == nil {
		plugins = make(map[string]Plugin)
	}
	if plugins[plugin.GetName()] != nil {
		panic("Plugin already exist: " + plugin.GetName())
	}
	plugins[plugin.GetName()] = plugin
}

func GetPlugin(plugin string) Plugin {
	if plugins == nil {
		return nil
	}

	return plugins[plugin]
}
