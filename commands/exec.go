package commands

import (
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"gitlab.com/ayufan/golang-cli-helpers"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/docker"
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/parallels"
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/virtualbox"
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/shell"
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/ssh"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type ExecCommand struct {
	common.RunnerSettings
	Job string
}

func (c *ExecCommand) runCommand(name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	result, err := cmd.Output()
	return string(result), err
}

func (c *ExecCommand) getCommands(commands interface{}) (string, error) {
	if lines, ok := commands.([]interface{}); ok {
		text := ""
		for _, line := range lines {
			if lineText, ok := line.(string); ok {
				text += lineText + "\n"
			} else {
				return "", errors.New("unsupported script")
			}
		}
		return text + "\n", nil
	} else if text, ok := commands.(string); ok {
		return text + "\n", nil
	} else if commands != nil {
		return "", errors.New("unsupported script")
	} else {
		return "", nil
	}
}

func (c *ExecCommand) supportedOption(key string, _ interface{}) bool {
	switch key {
	case "image", "services", "artifacts", "cache":
		return true
	default:
		return false
	}
}

func (c *ExecCommand) parseYaml(job string, build *common.GetBuildResponse) error {
	data, err := ioutil.ReadFile(".gitlab-ci.yml")
	if err != nil {
		return err
	}

	// parse gitlab-ci.yml
	config := make(map[string]interface{})
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return err
	}

	// get job
	jobConfig, ok := config[job].(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("no job named %q", job)
	}

	// get before_script
	beforeScript, err := c.getCommands(config["before_script"])
	if err != nil {
		return err
	}
	build.Commands = beforeScript

	// get script
	script, err := c.getCommands(jobConfig["script"])
	if err != nil {
		return err
	} else if jobConfig["script"] == nil {
		return fmt.Errorf("missing 'script' for %q", job)
	}
	build.Commands += script

	// parse variables
	if variables, ok := config["variables"].(map[interface{}]interface{}); ok {
		for key, value := range variables {
			if valueText, ok := value.(string); ok {
				build.Variables = append(build.Variables, common.BuildVariable{
					Key:    key.(string),
					Value:  valueText,
					Public: true,
				})
			} else {
				return fmt.Errorf("invalid value for variable %q", key)
			}
		}
	} else if config["variables"] != nil {
		return errors.New("unsupported variables")
	}

	build.Options = make(common.BuildOptions)

	// parse global options
	for key, value := range config {
		if c.supportedOption(key, value) {
			build.Options[key] = value
		}
	}

	// parse job options
	for key, value := range jobConfig {
		if c.supportedOption(key.(string), value) {
			build.Options[key.(string)] = value
		}
	}

	build.Name = job

	if stage, ok := jobConfig["stage"].(string); ok {
		build.Stage = stage
	} else {
		build.Stage = "test"
	}
	return nil
}

func (c *ExecCommand) Execute(context *cli.Context) {
	wd, err := os.Getwd()
	if err != nil {
		logrus.Fatalln(err)
	}

	switch len(context.Args()) {
	case 1:
		c.Job = context.Args().Get(0)
	default:
		cli.ShowSubcommandHelp(context)
		os.Exit(1)
		return
	}

	c.Executor = context.Command.Name

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	abortSignal := make(chan os.Signal)
	doneSignal := make(chan int, 1)

	go func() {
		interrupt := <-signals

		// request stop, but wait for force exit
		for interrupt == syscall.SIGQUIT {
			logrus.Warningln("Requested quit, waiting for builds to finish")
			interrupt = <-signals
		}

		logrus.Warningln("Requested exit:", interrupt)

		go func() {
			for {
				abortSignal <- interrupt
			}
		}()

		select {
		case newSignal := <-signals:
			logrus.Fatalln("forced exit:", newSignal)
		case <-time.After(common.ShutdownTimeout * time.Second):
			logrus.Fatalln("shutdown timedout")
		case <-doneSignal:
		}
	}()

	// Add self-volume to docker
	if c.RunnerSettings.Docker == nil {
		c.RunnerSettings.Docker = &common.DockerConfig{}
	}
	c.RunnerSettings.Docker.Volumes = append(c.RunnerSettings.Docker.Volumes, wd+":"+wd+":ro")

	// Check if we have uncomitted changes
	_, err = c.runCommand("git", "diff", "--quiet", "HEAD")
	if err != nil {
		logrus.Warningln("You most probably have uncommitted changes.")
		logrus.Warningln("These changes will not be tested.")
	}

	// Parse Git settings
	sha, err := c.runCommand("git", "rev-parse", "HEAD")
	if err != nil {
		logrus.Fatalln(err)
	}

	beforeSha, err := c.runCommand("git", "rev-parse", "HEAD~1")
	if err != nil {
		beforeSha = "0000000000000000000000000000000000000000"
	}

	refName, err := c.runCommand("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		logrus.Fatalln(err)
	}

	newBuild := common.Build{
		GetBuildResponse: common.GetBuildResponse{
			ID:            1,
			ProjectID:     1,
			RepoURL:       wd,
			Commands:      "",
			Sha:           strings.TrimSpace(sha),
			RefName:       strings.TrimSpace(refName),
			BeforeSha:     strings.TrimSpace(beforeSha),
			AllowGitFetch: false,
			Timeout:       30 * 60,
			Token:         "",
			Name:          "",
			Stage:         "",
			Tag:           false,
		},
		Runner: &common.RunnerConfig{
			RunnerSettings: c.RunnerSettings,
		},
		BuildAbort: abortSignal,
		Network:    nil,
	}

	err = c.parseYaml(c.Job, &newBuild.GetBuildResponse)
	if err != nil {
		logrus.Fatalln(err)
	}

	newBuild.AssignID()

	err = newBuild.Run(&common.Config{})
	if err != nil {
		logrus.Fatalln(err)
	}
}

func init() {
	cmd := &ExecCommand{}

	flags := clihelpers.GetFlagsFromStruct(cmd)
	cliCmd := cli.Command{
		Name:  "exec",
		Usage: "execute a build locally",
	}

	for _, executor := range common.GetExecutors() {
		subCmd := cli.Command{
			Name:   executor,
			Usage:  "use " + executor + " executor",
			Action: cmd.Execute,
			Flags:  flags,
		}
		cliCmd.Subcommands = append(cliCmd.Subcommands, subCmd)
	}

	common.RegisterCommand(cliCmd)
}
