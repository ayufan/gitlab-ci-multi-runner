package src

import (
	"github.com/codegangsta/cli"

	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
)

func runMulti(c *cli.Context) {
	config := Config{}

	if _, err := toml.DecodeFile(c.String("config"), &config); err != nil {
		panic(err)
	}
	log.Println("Starting multi-runner from", c.String("config"), "...")
}
