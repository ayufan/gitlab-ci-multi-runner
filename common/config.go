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
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/docker"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/ssh"
	"path/filepath"
)

type DockerPullPolicy string

const (
	DockerPullPolicyAlways       DockerPullPolicy = "always"
	DockerPullPolicyNever                         = "never"
	DockerPullPolicyIfNotPresent                  = "if-not-present"
)

// Get returns one of the predefined values or returns an error if the value can't match the predefined
func (p DockerPullPolicy) Get() (DockerPullPolicy, error) {
	// Default policy is always
	if p == "" {
		return DockerPullPolicyAlways, nil
	}

	// Verify pull policy
	if p != DockerPullPolicyNever &&
		p != DockerPullPolicyIfNotPresent &&
		p != DockerPullPolicyAlways {
		return "", fmt.Errorf("unsupported docker-pull-policy: %v", p)
	}
	return p, nil
}

type DockerConfig struct {
	docker_helpers.DockerCredentials
	Hostname               string           `toml:"hostname,omitempty" json:"hostname" long:"hostname" env:"DOCKER_HOSTNAME" description:"Custom container hostname"`
	Image                  string           `toml:"image" json:"image" long:"image" env:"DOCKER_IMAGE" description:"Docker image to be used"`
	CPUSetCPUs             string           `toml:"cpuset_cpus,omitempty" json:"cpuset_cpus" long:"cpuset-cpus" env:"DOCKER_CPUSET_CPUS" description:"String value containing the cgroups CpusetCpus to use"`
	DNS                    []string         `toml:"dns,omitempty" json:"dns" long:"dns" env:"DOCKER_DNS" description:"A list of DNS servers for the container to use"`
	DNSSearch              []string         `toml:"dns_search,omitempty" json:"dns_search" long:"dns-search" env:"DOCKER_DNS_SEARCH" description:"A list of DNS search domains"`
	Privileged             bool             `toml:"privileged,omitzero" json:"privileged" long:"privileged" env:"DOCKER_PRIVILEGED" description:"Give extended privileges to container"`
	CapAdd                 []string         `toml:"cap_add" json:"cap_add" long:"cap-add" env:"DOCKER_CAP_ADD" description:"Add Linux capabilities"`
	CapDrop                []string         `toml:"cap_drop" json:"cap_drop" long:"cap-drop" env:"DOCKER_CAP_DROP" description:"Drop Linux capabilities"`
	Devices                []string         `toml:"devices" json:"devices" long:"devices" env:"DOCKER_DEVICES" description:"Add a host device to the container"`
	DisableCache           bool             `toml:"disable_cache,omitzero" json:"disable_cache" long:"disable-cache" env:"DOCKER_DISABLE_CACHE" description:"Disable all container caching"`
	Volumes                []string         `toml:"volumes,omitempty" json:"volumes" long:"volumes" env:"DOCKER_VOLUMES" description:"Bind mount a volumes"`
	CacheDir               string           `toml:"cache_dir,omitempty" json:"cache_dir" long:"cache-dir" env:"DOCKER_CACHE_DIR" description:"Directory where to store caches"`
	ExtraHosts             []string         `toml:"extra_hosts,omitempty" json:"extra_hosts" long:"extra-hosts" env:"DOCKER_EXTRA_HOSTS" description:"Add a custom host-to-IP mapping"`
	NetworkMode            string           `toml:"network_mode,omitempty" json:"network_mode" long:"network-mode" env:"DOCKER_NETWORK_MODE" description:"Add container to a custom network"`
	Links                  []string         `toml:"links,omitempty" json:"links" long:"links" env:"DOCKER_LINKS" description:"Add link to another container"`
	Services               []string         `toml:"services,omitempty" json:"services" long:"services" env:"DOCKER_SERVICES" description:"Add service that is started with container"`
	WaitForServicesTimeout int              `toml:"wait_for_services_timeout,omitzero" json:"wait_for_services_timeout" long:"wait-for-services-timeout" env:"DOCKER_WAIT_FOR_SERVICES_TIMEOUT" description:"How long to wait for service startup"`
	AllowedImages          []string         `toml:"allowed_images,omitempty" json:"allowed_images" long:"allowed-images" env:"DOCKER_ALLOWED_IMAGES" description:"Whitelist allowed images"`
	AllowedServices        []string         `toml:"allowed_services,omitempty" json:"allowed_services" long:"allowed-services" env:"DOCKER_ALLOWED_SERVICES" description:"Whitelist allowed services"`
	PullPolicy             DockerPullPolicy `toml:"pull_policy,omitempty" json:"pull_policy" long:"pull-policy" env:"DOCKER_PULL_POLICY" description:"Image pull policy: never, if-not-present, always"`
}

type DockerMachine struct {
	IdleCount      int      `long:"idle-nodes" env:"MACHINE_IDLE_COUNT" description:"Maximum idle machines"`
	IdleTime       int      `toml:"IdleTime,omitzero" long:"idle-time" env:"MACHINE_IDLE_TIME" description:"Minimum time after node can be destroyed"`
	MaxBuilds      int      `toml:"MaxBuilds,omitzero" long:"max-builds" env:"MACHINE_MAX_BUILDS" description:"Maximum number of builds processed by machine"`
	MachineDriver  string   `long:"machine-driver" env:"MACHINE_DRIVER" description:"The driver to use when creating machine"`
	MachineName    string   `long:"machine-name" env:"MACHINE_NAME" description:"The template for machine name (needs to include %s)"`
	MachineOptions []string `long:"machine-options" env:"MACHINE_OPTIONS" description:"Additional machine creation options"`
}

type ParallelsConfig struct {
	BaseName         string `toml:"base_name" json:"base_name" long:"base-name" env:"PARALLELS_BASE_NAME" description:"VM name to be used"`
	TemplateName     string `toml:"template_name,omitempty" json:"template_name" long:"template-name" env:"PARALLELS_TEMPLATE_NAME" description:"VM template to be created"`
	DisableSnapshots bool   `toml:"disable_snapshots,omitzero" json:"disable_snapshots" long:"disable-snapshots" env:"PARALLELS_DISABLE_SNAPSHOTS" description:"Disable snapshoting to speedup VM creation"`
}

type VirtualBoxConfig struct {
	BaseName         string `toml:"base_name" json:"base_name" long:"base-name" env:"VIRTUALBOX_BASE_NAME" description:"VM name to be used"`
	DisableSnapshots bool   `toml:"disable_snapshots,omitzero" json:"disable_snapshots" long:"disable-snapshots" env:"VIRTUALBOX_DISABLE_SNAPSHOTS" description:"Disable snapshoting to speedup VM creation"`
}

type RunnerCredentials struct {
	URL       string `toml:"url" json:"url" short:"u" long:"url" env:"CI_SERVER_URL" required:"true" description:"Runner URL"`
	Token     string `toml:"token" json:"token" short:"t" long:"token" env:"CI_SERVER_TOKEN" required:"true" description:"Runner token"`
	TLSCAFile string `toml:"tls-ca-file,omitempty" json:"tls-ca-file" long:"tls-ca-file" env:"CI_SERVER_TLS_CA_FILE" description:"File containing the certificates to verify the peer when using HTTPS"`
}

type CacheConfig struct {
	Type           string `toml:"Type,omitempty" long:"type" env:"CACHE_TYPE" description:"Select caching method: s3, to use S3 buckets"`
	ServerAddress  string `toml:"ServerAddress,omitempty" long:"s3-server-address" env:"S3_SERVER_ADDRESS" description:"S3 Server Address"`
	AccessKey      string `toml:"AccessKey,omitempty" long:"s3-access-key" env:"S3_ACCESS_KEY" description:"S3 Access Key"`
	SecretKey      string `toml:"SecretKey,omitempty" long:"s3-secret-key" env:"S3_SECRET_KEY" description:"S3 Secret Key"`
	BucketName     string `toml:"BucketName,omitempty" long:"s3-bucket-name" env:"S3_BUCKET_NAME" description:"S3 bucket name"`
	BucketLocation string `toml:"BucketLocation,omitempty" long:"s3-bucket-location" env:"S3_BUCKET_LOCATION" description:"S3 location"`
	Insecure       bool   `toml:"Insecure,omitempty" long:"s3-insecure" env:"S3_CACHE_INSECURE" description:"Use insecure mode (without https)"`
}

type RunnerSettings struct {
	Executor  string `toml:"executor" json:"executor" long:"executor" env:"RUNNER_EXECUTOR" required:"true" description:"Select executor, eg. shell, docker, etc."`
	BuildsDir string `toml:"builds_dir,omitempty" json:"builds_dir" long:"builds-dir" env:"RUNNER_BUILDS_DIR" description:"Directory where builds are stored"`
	CacheDir  string `toml:"cache_dir,omitempty" json:"cache_dir" long:"cache-dir" env:"RUNNER_CACHE_DIR" description:"Directory where build cache is stored"`

	Environment []string `toml:"environment,omitempty" json:"environment" long:"env" env:"RUNNER_ENV" description:"Custom environment variables injected to build environment"`

	Shell string `toml:"shell,omitempty" json:"shell" long:"shell" env:"RUNNER_SHELL" description:"Select bash, cmd or powershell"`

	SSH        *ssh.Config       `toml:"ssh" json:"ssh" group:"ssh executor" namespace:"ssh"`
	Docker     *DockerConfig     `toml:"docker" json:"docker" group:"docker executor" namespace:"docker"`
	Parallels  *ParallelsConfig  `toml:"parallels" json:"parallels" group:"parallels executor" namespace:"parallels"`
	VirtualBox *VirtualBoxConfig `toml:"virtualbox" json:"virtualbox" group:"virtualbox executor" namespace:"virtualbox"`
	Cache      *CacheConfig      `toml:"cache" json:"cache" group:"cache configuration" namespace:"cache"`
	Machine    *DockerMachine    `toml:"machine" json:"machine" group:"docker machine provider" namespace:"machine"`
}

type RunnerConfig struct {
	Name        string `toml:"name" json:"name" short:"name" long:"description" env:"RUNNER_NAME" description:"Runner name"`
	Limit       int    `toml:"limit,omitzero" json:"limit" long:"limit" env:"RUNNER_LIMIT" description:"Maximum number of builds processed by this runner"`
	OutputLimit int    `toml:"output_limit,omitzero" long:"output-limit" env:"RUNNER_OUTPUT_LIMIT" description:"Maximum build trace size in kilobytes"`

	RunnerCredentials
	RunnerSettings
}

type Config struct {
	Concurrent    int             `toml:"concurrent" json:"concurrent"`
	CheckInterval int             `toml:"check_interval" json:"check_interval" description:"Define active checking interval of jobs"`
	User          string          `toml:"user,omitempty" json:"user"`
	Runners       []*RunnerConfig `toml:"runners" json:"runners"`
	SentryDSN     *string         `toml:"sentry_dsn"`
	ModTime       time.Time       `toml:"-"`
	Loaded        bool            `toml:"-"`
}

func (c *RunnerCredentials) ShortDescription() string {
	return helpers.ShortenToken(c.Token)
}

func (c *RunnerCredentials) UniqueID() string {
	return c.URL + c.Token
}

func (c *RunnerCredentials) Log() *log.Entry {
	if c.ShortDescription() != "" {
		return log.WithField("runner", c.ShortDescription())
	}
	return log.WithFields(log.Fields{})
}

func (c *RunnerConfig) String() string {
	return fmt.Sprintf("%v url=%v token=%v executor=%v", c.Name, c.URL, c.Token, c.Executor)
}

func (c *RunnerConfig) GetVariables() BuildVariables {
	var variables BuildVariables

	for _, environment := range c.Environment {
		if variable, err := ParseVariable(environment); err == nil {
			variable.Internal = true
			variables = append(variables, variable)
		}
	}

	return variables
}

func NewConfig() *Config {
	return &Config{
		Concurrent: 1,
	}
}

func (c *Config) StatConfig(configFile string) error {
	_, err := os.Stat(configFile)
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) LoadConfig(configFile string) error {
	info, err := os.Stat(configFile)

	// permission denied is soft error
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	if _, err = toml.DecodeFile(configFile, c); err != nil {
		return err
	}

	c.ModTime = info.ModTime()
	c.Loaded = true
	return nil
}

func (c *Config) SaveConfig(configFile string) error {
	var newConfig bytes.Buffer
	newBuffer := bufio.NewWriter(&newConfig)

	if err := toml.NewEncoder(newBuffer).Encode(c); err != nil {
		log.Fatalf("Error encoding TOML: %s", err)
		return err
	}

	if err := newBuffer.Flush(); err != nil {
		return err
	}

	// create directory to store configuration
	os.MkdirAll(filepath.Dir(configFile), 0700)

	// write config file
	if err := ioutil.WriteFile(configFile, newConfig.Bytes(), 0600); err != nil {
		return err
	}

	c.Loaded = true
	return nil
}

func (c *Config) GetCheckInterval() time.Duration {
	if c.CheckInterval > 0 {
		return time.Duration(c.CheckInterval) * time.Second
	}
	return CheckInterval
}
