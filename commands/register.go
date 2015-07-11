package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/ssh"
)

type RegistrationContext struct {
	*cli.Context
	configFile string
	config     *common.Config
	reader     *bufio.Reader
}

func (s *RegistrationContext) ask(key, prompt string, allowEmptyOptional ...bool) string {
	allowEmpty := len(allowEmptyOptional) > 0 && allowEmptyOptional[0]

	result := s.String(key)
	result = strings.TrimSpace(result)

	if s.Bool("non-interactive") || prompt == "" {
		if result == "" && !allowEmpty {
			err := fmt.Errorf("The '%s' needs to be entered", key)
			panic(err)
		}
		return result
	}

	for {
		println(prompt)
		if result != "" {
			print("["+result, "]: ")
		}

		data, _, err := s.reader.ReadLine()
		if err != nil {
			panic(err)
		}
		newResult := string(data)
		newResult = strings.TrimSpace(newResult)

		if newResult != "" {
			return newResult
		}

		if allowEmpty || result != "" {
			return result
		}
	}
}

func (s *RegistrationContext) askExecutor() string {
	for {
		names := common.GetExecutors()
		executors := strings.Join(names, ", ")
		result := s.ask("executor", "Please enter the executor: "+executors+":", true)
		if common.GetExecutor(result) != nil {
			return result
		}
	}
}

func (s *RegistrationContext) askForDockerService(service string, dockerConfig *common.DockerConfig) bool {
	for {
		result := s.ask("docker-"+service, "If you want to enable "+service+" please enter version (X.Y) or enter latest?", true)
		if len(result) == 0 {
			return false
		}
		if result != "latest" {
			_, err := strconv.ParseFloat(result, 32)
			if err != nil {
				println("Invalid version specified", err)
				continue
			}
		}
		dockerConfig.Services = append(dockerConfig.Services, service+":"+result)
		return true
	}
}

func (s *RegistrationContext) askDocker(runnerConfig *common.RunnerConfig) {
	dockerConfig := &common.DockerConfig{}
	dockerConfig.Image = s.ask("docker-image", "Please enter the Docker image (eg. ruby:2.1):")
	dockerConfig.Privileged = s.Bool("docker-privileged")

	if s.askForDockerService("mysql", dockerConfig) {
		runnerConfig.Environment = append(runnerConfig.Environment, "MYSQL_ALLOW_EMPTY_PASSWORD=1")
	}

	s.askForDockerService("postgres", dockerConfig)
	s.askForDockerService("redis", dockerConfig)
	s.askForDockerService("mongo", dockerConfig)

	dockerConfig.Volumes = append(dockerConfig.Volumes, "/cache")

	runnerConfig.Docker = dockerConfig
}

func (s *RegistrationContext) askParallels(runnerConfig *common.RunnerConfig) {
	parallelsConfig := &common.ParallelsConfig{}
	parallelsConfig.BaseName = s.ask("parallels-vm", "Please enter the Parallels VM (eg. my-vm):")
	runnerConfig.Parallels = parallelsConfig
}

func (s *RegistrationContext) askSSH(runnerConfig *common.RunnerConfig, serverless bool) {
	runnerConfig.SSH = &ssh.Config{}
	if !serverless {
		if host := s.ask("ssh-host", "Please enter the SSH server address (eg. my.server.com):"); host != "" {
			runnerConfig.SSH.Host = &host
		}
		if port := s.ask("ssh-port", "Please enter the SSH server port (eg. 22):", true); port != "" {
			runnerConfig.SSH.Port = &port
		}
	}
	if user := s.ask("ssh-user", "Please enter the SSH user (eg. root):"); user != "" {
		runnerConfig.SSH.User = &user
	}
	if password := s.ask("ssh-password", "Please enter the SSH password (eg. docker.io):", true); password != "" {
		runnerConfig.SSH.Password = &password
	}
	if identityFile := s.ask("ssh-identity-file", "Please enter path to SSH identity file (eg. /home/user/.ssh/id_rsa):", true); identityFile != "" {
		runnerConfig.SSH.IdentityFile = &identityFile
	}
}

func (s *RegistrationContext) loadConfig() {
	err := s.config.LoadConfig(s.configFile)
	if err != nil {
		panic(err)
	}
}

func (s *RegistrationContext) addRunner(runner *common.RunnerConfig) {
	s.config.Runners = append(s.config.Runners, runner)
}

func (s *RegistrationContext) saveConfig() {
	err := s.config.SaveConfig(s.configFile)
	if err != nil {
		panic(err)
	}
}

func (s *RegistrationContext) askRunner() common.RunnerConfig {
	url := s.ask("url", "Please enter the gitlab-ci coordinator URL (e.g. http://gitlab-ci.org:3000/):")
	registrationToken := s.ask("registration-token", "Please enter the gitlab-ci token for this runner:")
	description := s.ask("description", "Please enter the gitlab-ci description for this runner:")
	tagList := s.String("tag-list")

	result := common.RegisterRunner(url, registrationToken, description, tagList)
	if result == nil {
		log.Fatalf("Failed to register this runner. Perhaps you are having network problems")
	}

	return common.RunnerConfig{
		URL:   url,
		Name:  description,
		Token: result.Token,
	}
}

func runRegister(c *cli.Context) {
	s := RegistrationContext{
		Context:    c,
		config:     common.NewConfig(),
		configFile: c.String("config"),
		reader:     bufio.NewReader(os.Stdin),
	}

	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("FATAL ERROR: %v", r)
		}
	}()

	s.loadConfig()

	runnerConfig := s.askRunner()

	if !c.Bool("leave-runner") {
		defer func() {
			if r := recover(); r != nil {
				common.DeleteRunner(runnerConfig.URL, runnerConfig.Token)
				// pass panic to next defer
				panic(r)
			}
		}()

		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)

		go func() {
			s := <-signals
			common.DeleteRunner(runnerConfig.URL, runnerConfig.Token)
			log.Fatalf("RECEIVED SIGNAL: %v", s)
		}()
	}

	runnerConfig.Executor = s.askExecutor()
	limit := c.Int("limit")
	runnerConfig.Limit = &limit

	if s.config.Concurrent < limit {
		log.Warningf("Specified limit (%d) larger then current concurrent limit (%d). Concurrent limit will not be enlarged.", limit, s.config.Concurrent)
	}

	switch runnerConfig.Executor {
	case "docker", "docker-ssh":
		s.askDocker(&runnerConfig)
	case "parallels":
		s.askParallels(&runnerConfig)
	}

	switch runnerConfig.Executor {
	case "ssh":
		s.askSSH(&runnerConfig, false)
	case "docker-ssh":
		s.askSSH(&runnerConfig, true)
	case "parallels":
		s.askSSH(&runnerConfig, true)
	}

	s.addRunner(&runnerConfig)
	s.saveConfig()

	log.Printf("Runner registered successfully. Feel free to start it, but if it's running already the config should be automatically reloaded!")
}

func getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

func init() {
	common.RegisterCommand(cli.Command{
		Name:   "register",
		Usage:  "register a new runner",
		Action: runRegister,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "c, config",
				Value:  "config.toml",
				Usage:  "Config file",
				EnvVar: "CONFIG_FILE",
			},
			cli.BoolFlag{
				Name:   "n, non-interactive",
				Usage:  "Run registration unattended",
				EnvVar: "REGISTER_NON_INTERACTIVE",
			},
			cli.BoolFlag{
				Name:   "leave-runner",
				Usage:  "Don't remove runner if registration fails",
				EnvVar: "REGISTER_LEAVE_RUNNER",
			},

			cli.StringFlag{
				Name:   "r, registration-token",
				Value:  "",
				Usage:  "Runner's registration token",
				EnvVar: "REGISTRATION_TOKEN",
			},
			cli.StringFlag{
				Name:   "u, url",
				Value:  "",
				Usage:  "Runner URL",
				EnvVar: "CI_SERVER_URL",
			},
			cli.StringFlag{
				Name:   "d, description",
				Value:  getHostname(),
				Usage:  "Runner's registration description",
				EnvVar: "RUNNER_DESCRIPTION",
			},
			cli.StringFlag{
				Name:   "t, tag-list",
				Value:  "",
				Usage:  "Runner's tag list separated by comma",
				EnvVar: "RUNNER_TAG_LIST",
			},

			cli.StringFlag{
				Name:   "e, executor",
				Value:  "shell",
				Usage:  "Select executor, eg. shell, docker, etc.",
				EnvVar: "RUNNER_EXECUTOR",
			},

			// Runner specific configuration
			cli.IntFlag{
				Name:   "limit",
				Value:  1,
				Usage:  "Specify number of concurrent jobs for this runner",
				EnvVar: "RUNNER_LIMIT",
			},

			// Docker specific configuration
			cli.StringFlag{
				Name:   "docker-image",
				Value:  "",
				Usage:  "Docker image to use (eg. ruby:2.1)",
				EnvVar: "DOCKER_IMAGE",
			},
			cli.BoolFlag{
				Name:   "docker-privileged",
				Usage:  "Run Docker containers in privileged mode (INSECURE)",
				EnvVar: "DOCKER_PRIVILEGED",
			},
			cli.StringFlag{
				Name:   "docker-mysql",
				Usage:  "MySQL version (or specify latest) to link as service Docker service",
				EnvVar: "DOCKER_MYSQL",
			},
			cli.StringFlag{
				Name:   "docker-postgres",
				Usage:  "PostgreSQL version (or specify latest) to link as service Docker service",
				EnvVar: "DOCKER_POSTGRES",
			},
			cli.StringFlag{
				Name:   "docker-mongo",
				Usage:  "MongoDB version (or specify latest) to link as service Docker service",
				EnvVar: "DOCKER_MONGO",
			},
			cli.StringFlag{
				Name:   "docker-redis",
				Usage:  "Redis version (or specify latest) to link as service Docker service",
				EnvVar: "DOCKER_REDIS",
			},

			// Parallels specific configuration
			cli.StringFlag{
				Name:   "parallels-vm",
				Usage:  "Parallels VM to use (eg. Ubuntu Linux)",
				EnvVar: "PARALLELS_VM",
			},

			// SSH remote specific configuration
			cli.StringFlag{
				Name:   "ssh-host",
				Usage:  "SSH server address (eg. my.server.com)",
				EnvVar: "SSH_HOST",
			},
			cli.StringFlag{
				Name:   "ssh-port",
				Usage:  "SSH server port (default. 22)",
				EnvVar: "SSH_PORT",
			},

			// Docker SSH & remote specific configuration
			cli.StringFlag{
				Name:   "ssh-user",
				Usage:  "SSH client user",
				EnvVar: "SSH_USER",
			},
			cli.StringFlag{
				Name:   "ssh-password",
				Usage:  "SSH client password",
				EnvVar: "SSH_PASSWORD",
			},
			cli.StringFlag{
				Name:   "ssh-identity-file",
				Usage:  "SSH identity file",
				EnvVar: "SSH_IDENTITY_FILE",
			},
		},
	})
}
