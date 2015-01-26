package src

import (
	"bufio"
	"log"
	"os"
	"strings"

	"github.com/codegangsta/cli"
)

func ask(r *bufio.Reader, prompt string, result *string) {
	for len(*result) == 0 {
		log.Println(prompt)
		data, _, err := r.ReadLine()
		if err != nil {
			panic(err)
		}
		*result = string(data)
		*result = strings.TrimSpace(*result)
	}
}

func askExecutor(r *bufio.Reader, result *string) {
	for {
		ask(r, "Please enter the executor: shell, docker, docker-ssh?", result)
		switch *result {
		case "shell", "docker", "docker-ssh":
			return
		}
	}
}

func askDocker(r *bufio.Reader, runner_config *RunnerConfig) {
	ask(r, "Please enter the Docker image (eg. ruby:2.1):", &runner_config.DockerImage)
}

func askSsh(r *bufio.Reader, runner_config *RunnerConfig) {
	ask(r, "Please enter the SSH user (eg. root):", &runner_config.SshUser)
	ask(r, "Please enter the SSH password (eg. docker.io):", &runner_config.SshPassword)
}

func setup(c *cli.Context) {
	log.SetFlags(0)

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
	hostName := c.String("hostname")

	bio := bufio.NewReader(os.Stdin)
	ask(bio, "Please enter the gitlab-ci coordinator URL (e.g. http://gitlab-ci.org:3000/ )", &url)
	ask(bio, "Please enter the gitlab-ci token for this runner", &registrationToken)
	ask(bio, "Please enter the gitlab-ci hostname for this runner", &hostName)

	result := RegisterRunner(url, registrationToken, hostName)
	if result == nil {
		log.Fatalf("Failed to register this runner. Perhaps your SSH key is invalid or you are having network problems")
	}

	runner_config := RunnerConfig{
		URL:   url,
		Name:  hostName,
		Token: result.Token,
	}

	askExecutor(bio, &runner_config.Executor)

	switch runner_config.Executor {
	case "shell":
	case "docker", "docker-ssh":
		askDocker(bio, &runner_config)
	}

	switch runner_config.Executor {
	case "docker-ssh":
		askSsh(bio, &runner_config)
	}

	config.Runners = append(config.Runners, &runner_config)

	err = config.SaveConfig(c.String("config"))
	if err != nil {
		panic(err)
	}

	log.Printf("Runner registered successfully. Feel free to start it!")
}
