package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"

	"github.com/ayufan/gitlab-ci-multi-runner/common"
	"github.com/ayufan/gitlab-ci-multi-runner/ssh"
)

type SetupContext struct {
	*cli.Context
	configFile string
	config     common.Config
	reader     *bufio.Reader
}

func (s *SetupContext) ask(key, prompt string, allow_empty_optional ...bool) string {
	allow_empty := len(allow_empty_optional) > 0 && allow_empty_optional[0]

	result := s.String(key)
	result = strings.TrimSpace(result)

	if s.Bool("non-interactive") || prompt == "" {
		if result == "" && !allow_empty {
			err := errors.New(fmt.Sprintf("The '%s' needs to be entered", key))
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
		new_result := string(data)
		new_result = strings.TrimSpace(new_result)

		if new_result != "" {
			return new_result
		}

		if allow_empty || result != "" {
			return result
		}
	}
}

func (s *SetupContext) askExecutor() string {
	for {
		names := common.GetExecutors()
		executors := strings.Join(names, ", ")
		result := s.ask("executor", "Please enter the executor: "+executors+":", true)
		if common.GetExecutor(result) != nil {
			return result
		}
	}
}

func (s *SetupContext) askForDockerService(service string, docker_config *common.DockerConfig) bool {
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
		docker_config.Services = append(docker_config.Services, service+":"+result)
		return true
	}
}

func (s *SetupContext) askDocker(runner_config *common.RunnerConfig) {
	docker_config := &common.DockerConfig{}
	docker_config.Image = s.ask("docker-image", "Please enter the Docker image (eg. ruby:2.1):")

	if s.askForDockerService("mysql", docker_config) {
		runner_config.Environment = append(runner_config.Environment, "MYSQL_ALLOW_EMPTY_PASSWORD=1")
	}

	s.askForDockerService("postgres", docker_config)
	s.askForDockerService("redis", docker_config)
	s.askForDockerService("mongodb", docker_config)

	docker_config.Volumes = append(docker_config.Volumes, "/cache")

	runner_config.Docker = docker_config
}

func (s *SetupContext) askParallels(runner_config *common.RunnerConfig) {
	parallels_config := &common.ParallelsConfig{}
	parallels_config.BaseName = s.ask("parallels-vm", "Please enter the Parallels VM (eg. my-vm):")
	runner_config.Parallels = parallels_config
}

func (s *SetupContext) askSsh(runner_config *common.RunnerConfig, serverless bool) {
	runner_config.Ssh = &ssh.SshConfig{}
	if !serverless {
		runner_config.Ssh.Host = s.ask("ssh-host", "Please enter the SSH server address (eg. my.server.com):")
		runner_config.Ssh.Port = s.ask("ssh-port", "Please enter the SSH server port (eg. 22):", true)
	}
	runner_config.Ssh.User = s.ask("ssh-user", "Please enter the SSH user (eg. root):")
	runner_config.Ssh.Password = s.ask("ssh-password", "Please enter the SSH password (eg. docker.io):")
}

func (s *SetupContext) touchConfig() {
	file, _ := os.OpenFile(s.configFile, os.O_APPEND|os.O_CREATE, 0600)
	if file != nil {
		file.Close()
	}
}

func (s *SetupContext) loadConfig() {
	err := s.config.LoadConfig(s.configFile)
	if err != nil {
		panic(err)
	}
}

func (s *SetupContext) addRunner(runner *common.RunnerConfig) {
	s.config.Runners = append(s.config.Runners, runner)
}

func (s *SetupContext) saveConfig() {
	err := s.config.SaveConfig(s.configFile)
	if err != nil {
		panic(err)
	}
}

func (s *SetupContext) askRunner() common.RunnerConfig {
	url := s.ask("url", "Please enter the gitlab-ci coordinator URL (e.g. http://gitlab-ci.org:3000/):")
	registrationToken := s.ask("registration-token", "Please enter the gitlab-ci token for this runner:")
	description := s.ask("description", "Please enter the gitlab-ci description for this runner:")
	tagList := s.ask("tag-list", "", true)

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

func runSetup(c *cli.Context) {
	s := SetupContext{
		Context:    c,
		configFile: c.String("config"),
		reader:     bufio.NewReader(os.Stdin),
	}

	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("FATAL ERROR: %v", r)
		}
	}()

	s.touchConfig()
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

	switch runnerConfig.Executor {
	case "docker", "docker-ssh":
		s.askDocker(&runnerConfig)
	case "parallels":
		s.askParallels(&runnerConfig)
	}

	switch runnerConfig.Executor {
	case "ssh":
		s.askSsh(&runnerConfig, false)
	case "docker-ssh":
		s.askSsh(&runnerConfig, true)
	case "parallels":
		s.askSsh(&runnerConfig, true)
	}

	s.addRunner(&runnerConfig)
	s.saveConfig()

	log.Printf("Runner registered successfully. Feel free to start it, but if it's running already the config should be automatically reloaded!")
}

func getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

var (
	CmdRunSetup = cli.Command{
		Name:      "setup",
		ShortName: "s",
		Usage:     "setup a new runner",
		Action:    runSetup,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "c, config",
				Value:  "config.toml",
				Usage:  "Config file",
				EnvVar: "CONFIG_FILE",
			},
			cli.BoolFlag{
				Name:   "n, non-interactive",
				Usage:  "Run setup unattended",
				EnvVar: "SETUP_NON_INTERACTIVE",
			},
			cli.BoolFlag{
				Name:   "leave-runner",
				Usage:  "Don't remove runner if setup fails",
				EnvVar: "SETUP_LEAVE_RUNNER",
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

			// Docker specific configuration
			cli.StringFlag{
				Name:   "docker-image",
				Value:  "",
				Usage:  "Docker image to use (eg. ruby:2.1)",
				EnvVar: "DOCKER_IMAGE",
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
				Name:   "docker-mongodb",
				Usage:  "MongoDB version (or specify latest) to link as service Docker service",
				EnvVar: "DOCKER_MONGODB",
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
				EnvVar: "SSH_USER",
			},
		},
	}
)
