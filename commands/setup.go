package commands

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"

	"github.com/ayufan/gitlab-ci-multi-runner/common"
	"github.com/ayufan/gitlab-ci-multi-runner/ssh"
)

func ask(r *bufio.Reader, prompt string, result *string, allow_empty ...bool) {
	for len(*result) == 0 {
		println(prompt)
		data, _, err := r.ReadLine()
		if err != nil {
			panic(err)
		}
		*result = string(data)
		*result = strings.TrimSpace(*result)

		if len(allow_empty) > 0 && allow_empty[0] && len(*result) == 0 {
			return
		}
	}
}

func askExecutor(r *bufio.Reader, result *string) {
	for {
		ask(r, "Please enter the executor: shell, docker, docker-ssh, ssh, parallels?", result)
		if common.GetExecutor(*result) != nil {
			return
		}
	}
}

func askForDockerService(r *bufio.Reader, service string, docker_config *common.DockerConfig) bool {
	for {
		var result string
		ask(r, "If you want to enable "+service+" please enter version (X.Y) or enter latest?", &result, true)
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

func askDocker(r *bufio.Reader, runner_config *common.RunnerConfig) {
	docker_config := &common.DockerConfig{}
	ask(r, "Please enter the Docker image (eg. ruby:2.1):", &docker_config.Image)

	if askForDockerService(r, "mysql", docker_config) {
		runner_config.Environment = append(runner_config.Environment, "MYSQL_ALLOW_EMPTY_PASSWORD=1")
	}

	askForDockerService(r, "postgres", docker_config)
	askForDockerService(r, "redis", docker_config)
	askForDockerService(r, "mongodb", docker_config)

	docker_config.Volumes = append(docker_config.Volumes, "/cache")

	runner_config.Docker = docker_config
}

func askParallels(r *bufio.Reader, runner_config *common.RunnerConfig) {
	parallels_config := &common.ParallelsConfig{}
	ask(r, "Please enter the Parallels VM (eg. my-vm):", &parallels_config.BaseName)
	runner_config.Parallels = parallels_config
}

func askSsh(r *bufio.Reader, runner_config *common.RunnerConfig, serverless bool) {
	runner_config.Ssh = &ssh.SshConfig{}
	if !serverless {
		ask(r, "Please enter the SSH server address (eg. my.server.com):", &runner_config.Ssh.Host)
		ask(r, "Please enter the SSH server port (eg. 22):", &runner_config.Ssh.Port, true)
	}
	ask(r, "Please enter the SSH user (eg. root):", &runner_config.Ssh.User)
	ask(r, "Please enter the SSH password (eg. docker.io):", &runner_config.Ssh.Password)
}

func runSetup(c *cli.Context) {
	file, err := os.OpenFile(c.String("config"), os.O_APPEND|os.O_CREATE, 0600)
	if file != nil {
		file.Close()
	}

	config := common.Config{}
	err = config.LoadConfig(c.String("config"))
	if err != nil {
		panic(err)
	}

	url := c.String("url")
	registrationToken := c.String("registration-token")
	description := c.String("description")
	tags := c.String("tag-list")

	bio := bufio.NewReader(os.Stdin)
	ask(bio, "Please enter the gitlab-ci coordinator URL (e.g. http://gitlab-ci.org:3000/ )", &url)
	ask(bio, "Please enter the gitlab-ci token for this runner", &registrationToken)
	ask(bio, "Please enter the gitlab-ci description for this runner", &description)
	// ask(bio, "Please enter the tag list separated by comma or leave it empty", &tags, true)

	result := common.RegisterRunner(url, registrationToken, description, tags)
	if result == nil {
		log.Fatalf("Failed to register this runner. Perhaps your SSH key is invalid or you are having network problems")
	}

	runner_config := common.RunnerConfig{
		URL:              url,
		Name:             description,
		Token:            result.Token,
		Executor:         c.String("executor"),
		CleanEnvironment: c.Bool("clean_environment"),
	}

	askExecutor(bio, &runner_config.Executor)

	switch runner_config.Executor {
	case "docker", "docker-ssh":
		askDocker(bio, &runner_config)
	case "parallels":
		askParallels(bio, &runner_config)
	}

	switch runner_config.Executor {
	case "ssh":
		askSsh(bio, &runner_config, false)
	case "docker-ssh":
		askSsh(bio, &runner_config, true)
	case "parallels":
		askSsh(bio, &runner_config, true)
	}

	config.Runners = append(config.Runners, &runner_config)

	err = config.SaveConfig(c.String("config"))
	if err != nil {
		panic(err)
	}

	log.Printf("Runner registered successfully. Feel free to start it, but if it's running already the config should be automatically reloaded!")
}

var (
	CmdRunSetup = cli.Command{
		Name:      "setup",
		ShortName: "s",
		Usage:     "setup a new runner",
		Action:    runSetup,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "registration-token",
				Value:  "",
				Usage:  "Runner's registration token",
				EnvVar: "REGISTRATION_TOKEN",
			},
			cli.StringFlag{
				Name:   "url",
				Value:  "",
				Usage:  "Runner URL",
				EnvVar: "CI_SERVER_URL",
			},
			cli.StringFlag{
				Name:   "description",
				Value:  "",
				Usage:  "Runner's registration description",
				EnvVar: "RUNNER_DESCRIPTION",
			},
			cli.StringFlag{
				Name:   "config",
				Value:  "config.toml",
				Usage:  "Config file",
				EnvVar: "CONFIG_FILE",
			},
			cli.StringFlag{
				Name:   "tag-list",
				Value:  "",
				Usage:  "Runner's tag list separated by comma",
				EnvVar: "RUNNER_TAG_LIST",
			},
			cli.StringFlag{
				Name:   "executor",
				Value:  "",
				Usage:  "Select executor, eg. shell, docker, etc.",
				EnvVar: "RUNNER_EXECUTOR",
			},
			cli.BoolFlag{
				Name:   "clean_environment",
				Usage:  "do not inherit any environment vars from parent process (default: false)",
				EnvVar: "RUNNER_CLEANENVIRONMENT",
			},
		},
	}
)
