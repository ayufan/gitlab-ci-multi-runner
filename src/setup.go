package src

import (
	"bufio"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
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
		ask(r, "Please enter the executor: shell, docker, docker-ssh, ssh?", result)
		switch *result {
		case "shell", "docker", "docker-ssh", "ssh":
			return
		}
	}
}

func askDocker(r *bufio.Reader, runner_config *RunnerConfig) {
	runner_config.Docker = &DockerConfig{}
	ask(r, "Please enter the Docker image (eg. ruby:2.1):", &runner_config.Docker.Image)
}

func askSsh(r *bufio.Reader, runner_config *RunnerConfig, serverless bool) {
	runner_config.Ssh = &SshConfig{}
	if !serverless {
		ask(r, "Please enter the SSH server address (eg. my.server.com):", &runner_config.Ssh.Host)
		ask(r, "Please enter the SSH server port (eg. 22):", &runner_config.Ssh.Port, true)
	}
	ask(r, "Please enter the SSH user (eg. root):", &runner_config.Ssh.User)
	ask(r, "Please enter the SSH password (eg. docker.io):", &runner_config.Ssh.Password)
}

func setup(c *cli.Context) {
	file, err := os.OpenFile(c.String("config"), os.O_APPEND|os.O_CREATE, 0600)
	if file != nil {
		file.Close()
	}

	config := Config{}
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

	result := RegisterRunner(url, registrationToken, description, tags)
	if result == nil {
		log.Fatalf("Failed to register this runner. Perhaps your SSH key is invalid or you are having network problems")
	}

	runner_config := RunnerConfig{
		URL:      url,
		Name:     description,
		Token:    result.Token,
		Executor: c.String("executor"),
	}

	askExecutor(bio, &runner_config.Executor)

	switch runner_config.Executor {
	case "docker", "docker-ssh":
		askDocker(bio, &runner_config)
	}

	switch runner_config.Executor {
	case "ssh":
		askSsh(bio, &runner_config, false)
	case "docker-ssh":
		askSsh(bio, &runner_config, true)
	}

	config.Runners = append(config.Runners, &runner_config)

	err = config.SaveConfig(c.String("config"))
	if err != nil {
		panic(err)
	}

	log.Printf("Runner registered successfully. Feel free to start it, but if it's running already the config should be automatically reloaded!")
}
