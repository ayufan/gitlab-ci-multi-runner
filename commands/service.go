package commands

import (
	log "github.com/Sirupsen/logrus"
	"github.com/ayufan/gitlab-ci-multi-runner/common"
	"github.com/codegangsta/cli"
	"github.com/kardianos/service"
	"os"
	"os/user"
	"runtime"
)

type ServiceLogHook struct {
	service.Logger
}

func (s *ServiceLogHook) Levels() []log.Level {
	return []log.Level{
		log.PanicLevel,
		log.FatalLevel,
		log.ErrorLevel,
		log.WarnLevel,
		log.InfoLevel,
	}
}

func (s *ServiceLogHook) Fire(e *log.Entry) error {
	switch e.Level {
	case log.PanicLevel, log.FatalLevel, log.ErrorLevel:
		s.Error(e.Message)
	case log.WarnLevel:
		s.Warning(e.Message)
	case log.InfoLevel:
		s.Info(e.Message)
	}
	return nil
}

type NullService struct {
}

func (n *NullService) Start(s service.Service) error {
	return nil
}

func (n *NullService) Stop(s service.Service) error {
	return nil
}

func RunServiceControl(c *cli.Context) {
	serviceName := c.String("service-name")
	displayName := c.String("service-name")
	if serviceName == "" {
		serviceName = "gitlab-ci-multi-runner"
		displayName = "GitLab-CI Multi-purpose Runner"
	}

	svcConfig := &service.Config{
		Name:        serviceName,
		DisplayName: displayName,
		Description: "Unofficial GitLab CI runner written in Go",
		Arguments:   []string{"run"},
		UserName:    c.String("user"),
	}

	if runtime.GOOS == "darwin" {
		svcConfig.UserService = true
		svcConfig.Option = service.KeyValue{
			"KeepAlive": true,
			"RunAtLoad": true,
			"Password":  c.String("password"),
		}
	}

	if wd := c.String("working-directory"); wd != "" {
		svcConfig.Arguments = append(svcConfig.Arguments, "--working-directory", wd)
	}

	if config := c.String("config"); config != "" {
		svcConfig.Arguments = append(svcConfig.Arguments, "--config", config)
	}

	s, err := service.New(&NullService{}, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	err = service.Control(s, c.Command.Name)
	if err != nil {
		log.Fatal(err)
	}
}

func getCurrentUserName() string {
	user, _ := user.Current()
	if user != nil {
		return user.Username
	}
	return ""
}

func getCurrentWorkingDirectory() string {
	dir, err := os.Getwd()
	if err == nil {
		return dir
	}
	return ""
}

func init() {
	flags := []cli.Flag{
		cli.StringFlag{
			Name:  "service-name, n",
			Value: "",
			Usage: "Use different names for different services",
		},
	}

	installFlags := flags
	installFlags = append(installFlags, cli.StringFlag{
		Name:  "working-directory, d",
		Value: getCurrentWorkingDirectory(),
		Usage: "Specify custom root directory where all data are stored",
	})
	installFlags = append(installFlags, cli.StringFlag{
		Name:  "config, c",
		Value: "config.toml",
		Usage: "Specify custom config file",
	})

	if runtime.GOOS != "darwin" {
		installFlags = append(installFlags, cli.StringFlag{
			Name:  "user, u",
			Value: getCurrentUserName(),
			Usage: "Specify user-name to secure the runner",
		})
	}

	if runtime.GOOS == "windows" {
		installFlags = append(installFlags, cli.StringFlag{
			Name:  "password, p",
			Value: "",
			Usage: "Specify user password to install service (required)",
		})
	}

	common.RegisterCommand(cli.Command{
		Name:   "install",
		Usage:  "install service",
		Action: RunServiceControl,
		Flags:  installFlags,
	})
	common.RegisterCommand(cli.Command{
		Name:   "uninstall",
		Usage:  "uninstall service",
		Action: RunServiceControl,
		Flags:  flags,
	})
	common.RegisterCommand(cli.Command{
		Name:   "start",
		Usage:  "start service",
		Action: RunServiceControl,
		Flags:  flags,
	})
	common.RegisterCommand(cli.Command{
		Name:   "stop",
		Usage:  "stop service",
		Action: RunServiceControl,
		Flags:  flags,
	})
	common.RegisterCommand(cli.Command{
		Name:   "restart",
		Usage:  "restart service",
		Action: RunServiceControl,
		Flags:  flags,
	})
}
