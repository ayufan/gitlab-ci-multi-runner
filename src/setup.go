package src

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/codegangsta/cli"
)

func setup(c *cli.Context) {
	log.SetFlags(0)

	config := Config{
		Concurrent: 1,
	}

	if _, err := os.Stat(flConfigFile.Value); err == nil {
		if _, err := toml.DecodeFile(flConfigFile.Value, &config); err != nil {
			panic(err)
		}
	}

	runner_config := RunnerConfig{
		URL:   c.String("URL"),
		Token: c.String("registration-token"),
		Limit: 1,
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

	config.Runners = append(config.Runners, runner_config)

	var new_config bytes.Buffer
	new_buffer := bufio.NewWriter(&new_config)

	if err := toml.NewEncoder(new_buffer).Encode(&config); err != nil {
		log.Fatalf("Error encoding TOML: %s", err)
	}

	if err := new_buffer.Flush(); err != nil {
		panic(err)
	}

	if err := ioutil.WriteFile(flConfigFile.Value, new_config.Bytes(), 0600); err != nil {
		panic(err)
	}

	log.Printf("Runner registered successfully. Feel free to start it!")
}
