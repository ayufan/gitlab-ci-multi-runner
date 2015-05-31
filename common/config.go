package common

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"time"

	"fmt"
	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/ssh"
	"runtime"
)

type DockerConfig struct {
	Host                   *string  `toml:"host" json:"host"`
	CertPath               *string  `toml:"tls_cert_path" json:"tls_cert_path"`
	Hostname               *string  `toml:"hostname" json:"hostname"`
	Image                  string   `toml:"image" json:"image"`
	Privileged             bool     `toml:"privileged" json:"privileged"`
	DisableCache           *bool    `toml:"disable_cache" json:"disable_cache"`
	Volumes                []string `toml:"volumes" json:"volumes"`
	CacheDir               *string  `toml:"cache_dir" json:"cache_dir"`
	Registry               *string  `toml:"registry" json:"registry"`
	ExtraHosts             []string `toml:"extra_hosts" json:"extra_hosts"`
	Links                  []string `toml:"links" json:"links"`
	Services               []string `toml:"services" json:"services"`
	WaitForServicesTimeout *int     `toml:"wait_for_services_timeout" json:"wait_for_services_timeout"`
}

type ParallelsConfig struct {
	BaseName         string  `toml:"base_name" json:"base_name"`
	TemplateName     *string `toml:"template_name" json:"template_name"`
	DisableSnapshots *bool   `toml:"disable_snapshots" json:"disable_snapshots"`
}

type RunnerConfig struct {
	Name      string  `toml:"name" json:"name"`
	URL       string  `toml:"url" json:"url"`
	Token     string  `toml:"token" json:"token"`
	Limit     *int    `toml:"limit" json:"limit"`
	Executor  string  `toml:"executor" json:"executor"`
	BuildsDir *string `toml:"builds_dir" json:"builds_dir"`

	Environment []string `toml:"environment" json:"environment"`

	Shell          *string `toml:"shell" json:"shell"`
	DisableVerbose *bool   `toml:"disable_verbose" json:"disable_verbose"`
	OutputLimit    *int    `toml:"output_limit"`

	SSH       *ssh.Config      `toml:"ssh" json:"ssh"`
	Docker    *DockerConfig    `toml:"docker" json:"docker"`
	Parallels *ParallelsConfig `toml:"parallels" json:"parallels"`
}

type BaseConfig struct {
	Concurrent int             `toml:"concurrent" json:"concurrent"`
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

func (c *RunnerConfig) String() string {
	return fmt.Sprintf("%v url=%v token=%v executor=%v", c.Name, c.URL, c.Token, c.Executor)
}

func NewConfig() *Config {
	return &Config{
		BaseConfig: BaseConfig{
			Concurrent: runtime.NumCPU(),
		},
	}
}

func (c *Config) LoadConfig(configFile string) error {
	info, err := os.Stat(configFile)
	if err != nil {
		return err
	}

	if _, err = toml.DecodeFile(configFile, &c.BaseConfig); err != nil {
		return err
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
