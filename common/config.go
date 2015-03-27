package common

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
	"github.com/ayufan/gitlab-ci-multi-runner/helpers"
	"github.com/ayufan/gitlab-ci-multi-runner/ssh"
)

type DockerConfig struct {
	Host                   string   `toml:"host" json:"host"`
	Hostname               string   `toml:"hostname" json:"hostname"`
	Image                  string   `toml:"image" json:"image"`
	Privileged             bool     `toml:"privileged" json:"privileged"`
	DisableCache           bool     `toml:"disable_cache" json:"disable_cache"`
	DisablePull            bool     `toml:"disable_pull" json:"disable_pull"`
	Volumes                []string `toml:"volumes" json:"volumes"`
	CacheDir               string   `toml:"cache_dir" json:"cache_dir"`
	Registry               string   `toml:"registry" json:"registry"`
	ExtraHosts             []string `toml:"extra_hosts" json:"extra_hosts"`
	Links                  []string `toml:"links" json:"links"`
	Services               []string `toml:"services" json:"services"`
	WaitForServicesTimeout *int     `toml:"wait_for_services_timeout" json:"wait_for_services_timeout"`
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

	CleanEnvironment bool     `toml:"clean_environment" json:"clean_environment"`
	Environment      []string `toml:"environment" json:"environment"`

	DisableVerbose bool `toml:"disable_verbose" json:"disable_verbose"`

	SSH       *ssh.Config      `toml:"ssh" json:"ssh"`
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
	return helpers.ShortenToken(c.Token)
}

func (c *RunnerConfig) UniqueID() string {
	return c.URL + c.Token
}

func (c *Config) LoadConfig(configFile string) error {
	info, err := os.Stat(configFile)
	if err != nil {
		return err
	}

	if _, err = toml.DecodeFile(configFile, &c.BaseConfig); err != nil {
		return err
	}

	if c.Concurrent == 0 {
		c.Concurrent = 1
	}

	c.ModTime = info.ModTime()
	return nil
}

func (c *Config) SaveConfig(configFile string) error {
	var newConfig bytes.Buffer
	newBuffer := bufio.NewWriter(&newConfig)

	if err := toml.NewEncoder(newBuffer).Encode(&c.BaseConfig); err != nil {
		log.Fatalf("Error encoding TOML: %s", err)
		return err
	}

	if err := newBuffer.Flush(); err != nil {
		return err
	}

	if err := ioutil.WriteFile(configFile, newConfig.Bytes(), 0600); err != nil {
		return err
	}

	return nil
}

func (c *Config) SetChdir() {
	if len(c.RootDir) > 0 {
		err := os.Chdir(c.RootDir)
		if err != nil {
			panic(err)
		}
	}
}
