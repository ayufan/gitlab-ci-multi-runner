package src

import (
	"bufio"
	"log"
	"os"
	"strings"

	"github.com/codegangsta/cli"
)

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

	runner_config := RunnerConfig{
		URL:   c.String("url"),
		Token: c.String("registration-token"),
	}

	bio := bufio.NewReader(os.Stdin)
	for len(runner_config.URL) == 0 {
		log.Printf("Please enter the gitlab-ci coordinator URL (e.g. http://gitlab-ci.org:3000/ )")
		data, _, err := bio.ReadLine()
		if err != nil {
			panic(err)
		}
		runner_config.URL = string(data)
		runner_config.URL = strings.TrimSpace(runner_config.URL)
	}

	for len(runner_config.Name) == 0 {
		log.Printf("Please enter the gitlab-ci hostname for this runner:")
		data, _, err := bio.ReadLine()
		if err != nil {
			panic(err)
		}
		runner_config.Name = string(data)
		runner_config.Name = strings.TrimSpace(runner_config.Name)
	}

	for len(runner_config.Token) == 0 {
		log.Printf("Please enter the gitlab-ci token for this runner:")
		data, _, err := bio.ReadLine()
		if err != nil {
			panic(err)
		}
		runner_config.Token = string(data)
		runner_config.Token = strings.TrimSpace(runner_config.Token)
	}

	result := RegisterRunner(runner_config)
	if result == nil {
		log.Fatalf("Failed to register this runner. Perhaps your SSH key is invalid or you are having network problems")
	}

	runner_config.Token = result.Token
	runner_config.DockerVolumes = []string{"/test", "/second"}

	config.Runners = append(config.Runners, &runner_config)

	err = config.SaveConfig(c.String("config"))
	if err != nil {
		panic(err)
	}

	log.Printf("Runner registered successfully. Feel free to start it!")
}
