package common

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
	"github.com/ayufan/gitlab-ci-multi-runner/ssh"
)

type DockerConfig struct {
	Host         string   `toml:"host" json:"host"`
	Hostname     string   `toml:"hostname" json:"hostname"`
	Image        string   `toml:"image" json:"image"`
	Privileged   bool     `toml:"privileged" json:"privileged"`
	DisableCache bool     `toml:"disable_cache" json:"disable_cache"`
	DisablePull  bool     `toml:"disable_pull" json:"disable_pull"`
	Volumes      []string `toml:"volumes" json:"volumes"`
	CacheDir     string   `toml:"cache_dir" json:"cache_dir"`
	Registry     string   `toml:"registry" json:"registry"`
	ExtraHosts   []string `toml:"extra_hosts" json:"extra_hosts"`
	Links        []string `toml:"links" json:"links"`
	Services     []string `toml:"services" json:"services"`
}

type ParallelsConfig struct {
	BaseName         string `toml:"base_name" json:"base_name"`
	TemplateName     string `toml:"template_name" json:"template_name"`
	DisableSnapshots bool   `toml:"disable_snapshots" json:"disable_snapshots"`
}

type RunnerConfig struct {
	Name      string `toml:"name" json:"name"`
	URL       string `toml:"url" json:"url"`
	Token     string `toml:"token" json:"token"`
	Limit     int    `toml:"limit" json:"limit"`
	Executor  string `toml:"executor" json:"executor"`
	BuildsDir string `toml:"builds_dir" json:"builds_dir"`

	CleanEnvironment  bool     `toml:"clean_environment" json:"clean_environment"`
	Environment       []string `toml:"environment" json:"environment"`

	ShellScript    string `toml:"shell_script" json:"shell_script"`
	DisableVerbose bool   `toml:"disable_verbose" json:"disable_verbose"`

	Ssh       *ssh.SshConfig   `toml:"ssh" json:"ssh"`
	Docker    *DockerConfig    `toml:"docker" json:"docker"`
	Parallels *ParallelsConfig `toml:"parallels" json:"parallels"`
}

type BaseConfig struct {
	Concurrent int             `toml:"concurrent" json:"concurrent"`
	RootDir    string          `toml:"root_dir" json:"root_dir"`
	Runners    []*RunnerConfig `toml:"runners" json:"runners"`
}

type Config struct {
	BaseConfig
	ModTime time.Time `json:"-"`
}

func (c *RunnerConfig) ShortDescription() string {
	return c.Token[0:8]
}

func (c *RunnerConfig) UniqueID() string {
	return c.URL + c.Token
}

func (config *Config) LoadConfig(config_file string) error {
	info, err := os.Stat(config_file)
	if err != nil {
		return err
	}

	if _, err = toml.DecodeFile(config_file, &config.BaseConfig); err != nil {
		return err
	}

	if config.Concurrent == 0 {
		config.Concurrent = 1
	}

	config.ModTime = info.ModTime()
	return nil
}

func (config *Config) SaveConfig(config_file string) error {
	var new_config bytes.Buffer
	new_buffer := bufio.NewWriter(&new_config)

	if err := toml.NewEncoder(new_buffer).Encode(&config.BaseConfig); err != nil {
		log.Fatalf("Error encoding TOML: %s", err)
		return err
	}

	if err := new_buffer.Flush(); err != nil {
		return err
	}

	if err := ioutil.WriteFile(config_file, new_config.Bytes(), 0600); err != nil {
		return err
	}

	return nil
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

			new_config := Config{}
			err := new_config.LoadConfig(config_file)
			if err != nil {
				log.Errorln("Failed to load config", err)
				continue
			}

			reload_config <- new_config
		}
	}
}

func (c *Config) SetChdir() {
	if len(c.RootDir) > 0 {
		err := os.Chdir(c.RootDir)
		if err != nil {
			panic(err)
		}
	}
}
