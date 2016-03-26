package commands

import (
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"gitlab.com/ayufan/golang-cli-helpers"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	// Force to load all executors, executes init() on them
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/docker"
	_ "gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors/shell"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
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

func (c *ExecCommand) buildVariables(configVariables interface{}) (buildVariables common.BuildVariables, err error) {
	if variables, ok := configVariables.(map[interface{}]interface{}); ok {
		for key, value := range variables {
			if valueText, ok := value.(string); ok {
				buildVariables = append(buildVariables, common.BuildVariable{
					Key:    key.(string),
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

func (c *ExecCommand) isJob(config map[string]interface{}, name string) (ok bool) {
	_, ok = helpers.GetMapKey(config, name, "plugin")
	if !ok {
		_, ok = helpers.GetMapKey(config, name, "script")
	}
	return
}

func (c *ExecCommand) buildOptions(config map[string]interface{},
	jobConfig map[interface{}]interface{}) (options common.BuildOptions, err error) {

	options = make(common.BuildOptions)

	// parse global options
	for key, value := range config {
		if c.isJob(config, key) {
			continue
		}
		if key == "variables" {
			continue
		}
		options[key] = value
	}

	// parse job options
	for key, value := range jobConfig {
		keyName := key.(string)
		if keyName == "stage" || keyName == "plugin" {
			continue
		}
		options[keyName] = value
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

	build.Variables, err = c.buildVariables(config["variables"])
	if err != nil {
		return err
	}

	build.Options, err = c.buildOptions(config, jobConfig)
	if err != nil {
		return err
	}

	if stage, ok := jobConfig["stage"].(string); ok {
		build.Stage = stage
	} else {
		build.Stage = "test"
	}

	if stage, ok := jobConfig["plugin"].(string); ok {
		build.Plugin = stage
	} else {
		build.Plugin = "script"
	}
	return nil
}

func (c *ExecCommand) createBuild(repoURL string) (build *common.Build, err error) {
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

	build = common.NewBuild(common.GetBuildResponse{
		ID:            1,
		ProjectID:     1,
		RepoURL:       repoURL,
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
		common.RunnerConfig{
			RunnerSettings: c.RunnerSettings,
		},
	)
	return
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
	build, err := c.createBuild(wd)
	if err != nil {
		logrus.Fatalln(err)
	}

	err = c.parseYaml(c.Job, &build.GetBuildResponse)
	if err != nil {
		logrus.Fatalln(err)
	}

	err = build.Run(nil, &stdoutTrace{}, abortSignal)
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
