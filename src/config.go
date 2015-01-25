package src

import (
	"os"
	"time"

	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
)

type RunnerConfig struct {
	Name      string `toml:"name",omitempty`
	URL       string `toml:"url"`
	Token     string `toml:"token"`
	Limit     int    `toml:"limit",omitempty`
	Executor  string `toml:"executor",omitempty`
	BuildsDir string `toml:"builds_dir",omitempty`
}

type Config struct {
	RootDir    string          `toml:"root_dir"`
	Concurrent int             `toml:"concurrent"`
	Runners    []*RunnerConfig `toml:"runners"`
}

func (c *RunnerConfig) GetBuildsDir() string {
	if len(c.BuildsDir) == 0 {
		return "tmp/builds"
	} else {
		return c.BuildsDir
	}
}

func (c *RunnerConfig) ShortDescription() string {
	return c.Token[0:8]
}

func LoadConfig(config_file string) (Config, time.Time, error) {
	config := Config{}

	info, err := os.Stat(config_file)
	if err != nil {
		return config, time.Time{}, err
	}

	if _, err = toml.DecodeFile(config_file, &config); err != nil {
		return config, info.ModTime(), err
	}

	if config.Concurrent == 0 {
		config.Concurrent = 1
	}

	if len(config.RootDir) > 0 {
		err = os.Chdir(config.RootDir)
		if err != nil {
			panic(err)
		}
	}

	return config, info.ModTime(), nil
}

func ReloadConfig(config_file string, config_time time.Time, reload_config chan Config) {
	for {
		time.Sleep(RELOAD_CONFIG_INTERVAL * time.Second)

		info, err := os.Stat(config_file)
		if err != nil {
			log.Errorln("Failed to stat config", err)
			continue
		}

		if config_time.Before(info.ModTime()) {
			config_time = info.ModTime()

			new_config, _, err := LoadConfig(config_file)
			if err != nil {
				log.Errorln("Failed to load config", err)
				continue
			}

			reload_config <- new_config
		}
	}
}
