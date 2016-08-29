package commands

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"gitlab.com/ayufan/golang-cli-helpers"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gopkg.in/yaml.v1"

	// Force to load all executors, executes init() on them
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/docker"
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/parallels"
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/shell"
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/ssh"
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/virtualbox"
)

type ExecCommand struct {
	common.RunnerSettings
	Job     string
	Timeout int `long:"timeout" description:"Job execution timeout (in seconds)"`
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
	}
	return "", nil
}

func (c *ExecCommand) supportedOption(key string, _ interface{}) bool {
	switch key {
	case "image", "services", "artifacts", "cache", "after_script":
		return true
	default:
		return false
	}
}

func (c *ExecCommand) buildCommands(configBeforeScript, jobScript interface{}) (commands string, err error) {
	// get before_script
	beforeScript, err := c.getCommands(configBeforeScript)
	if err != nil {
		return
	}
	commands += beforeScript

	// get script
	script, err := c.getCommands(jobScript)
	if err != nil {
		return
	} else if jobScript == nil {
		err = fmt.Errorf("missing 'script' for job")
		return
	}
	commands += script
	return
}

func (c *ExecCommand) buildVariables(configVariables interface{}) (buildVariables common.BuildVariables, err error) {
	if variables, ok := configVariables.(map[string]interface{}); ok {
		for key, value := range variables {
			if valueText, ok := value.(string); ok {
				buildVariables = append(buildVariables, common.BuildVariable{
					Key:    key,
					Value:  valueText,
					Public: true,
				})
			} else {
				err = fmt.Errorf("invalid value for variable %q", key)
			}
		}
	} else if configVariables != nil {
		err = errors.New("unsupported variables")
	}
	return
}

func (c *ExecCommand) buildGlobalAndJobVariables(global, job interface{}) (buildVariables common.BuildVariables, err error) {
	buildVariables, err = c.buildVariables(global)
	if err != nil {
		return
	}

	jobVariables, err := c.buildVariables(job)
	if err != nil {
		return
	}

	buildVariables = append(buildVariables, jobVariables...)
	return
}

func (c *ExecCommand) buildOptions(config, jobConfig common.BuildOptions) (options common.BuildOptions, err error) {
	options = make(common.BuildOptions)

	// parse global options
	for key, value := range config {
		if c.supportedOption(key, value) {
			options[key] = value
		}
	}

	// parse job options
	for key, value := range jobConfig {
		if c.supportedOption(key, value) {
			options[key] = value
		}
	}
	return
}

func (c *ExecCommand) parseYaml(job string, build *common.GetBuildResponse) error {
	data, err := ioutil.ReadFile(".gitlab-ci.yml")
	if err != nil {
		return err
	}

	build.Name = job

	// parse gitlab-ci.yml
	config := make(common.BuildOptions)
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return err
	}

	err = config.Sanitize()
	if err != nil {
		return err
	}

	// get job
	jobConfig, ok := config.GetSubOptions(job)
	if !ok {
		return fmt.Errorf("no job named %q", job)
	}

	build.Commands, err = c.buildCommands(config["before_script"], jobConfig["script"])
	if err != nil {
		return err
	}

	build.Variables, err = c.buildGlobalAndJobVariables(config["variables"], jobConfig["variables"])
	if err != nil {
		return err
	}

	build.Options, err = c.buildOptions(config, jobConfig)
	if err != nil {
		return err
	}

	if stage, ok := jobConfig.GetString("stage"); ok {
		build.Stage = stage
	} else {
		build.Stage = "test"
	}
	return nil
}

func (c *ExecCommand) createBuild(repoURL string, abortSignal chan os.Signal) (build *common.Build, err error) {
	// Check if we have uncommitted changes
	_, err = c.runCommand("git", "diff", "--quiet", "HEAD")
	if err != nil {
		logrus.Warningln("You most probably have uncommitted changes.")
		logrus.Warningln("These changes will not be tested.")
	}

	// Parse Git settings
	sha, err := c.runCommand("git", "rev-parse", "HEAD")
	if err != nil {
		return
	}

	beforeSha, err := c.runCommand("git", "rev-parse", "HEAD~1")
	if err != nil {
		beforeSha = "0000000000000000000000000000000000000000"
	}

	refName, err := c.runCommand("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return
	}

	build = &common.Build{
		GetBuildResponse: common.GetBuildResponse{
			ID:            1,
			ProjectID:     1,
			RepoURL:       repoURL,
			Commands:      "",
			Sha:           strings.TrimSpace(sha),
			RefName:       strings.TrimSpace(refName),
			BeforeSha:     strings.TrimSpace(beforeSha),
			AllowGitFetch: false,
			Timeout:       c.getTimeout(),
			Token:         "",
			Name:          "",
			Stage:         "",
			Tag:           false,
		},
		Runner: &common.RunnerConfig{
			RunnerSettings: c.RunnerSettings,
		},
		SystemInterrupt: abortSignal,
	}
	return
}

func (c *ExecCommand) getTimeout() int {
	if c.Timeout > 0 {
		return c.Timeout
	}

	return common.DefaultExecTimeout
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

	abortSignal := make(chan os.Signal)
	doneSignal := make(chan int, 1)

	go waitForInterrupts(nil, abortSignal, doneSignal)

	// Add self-volume to docker
	if c.RunnerSettings.Docker == nil {
		c.RunnerSettings.Docker = &common.DockerConfig{}
	}
	c.RunnerSettings.Docker.Volumes = append(c.RunnerSettings.Docker.Volumes, wd+":"+wd+":ro")

	// Create build
	build, err := c.createBuild(wd, abortSignal)
	if err != nil {
		logrus.Fatalln(err)
	}

	err = c.parseYaml(c.Job, &build.GetBuildResponse)
	if err != nil {
		logrus.Fatalln(err)
	}

	err = build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
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
