package commands

import (
	log "github.com/Sirupsen/logrus"
	service "github.com/ayufan/golang-kardianos-service"
	"github.com/codegangsta/cli"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"runtime"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"os"
)

const (
	defaultServiceName = "gitlab-runner"
	defaultDisplayName = "GitLab Runner"
	defaultDescription = "GitLab Runner"
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

func runServiceInstall(s service.Service, c *cli.Context) error {
	if user := c.String("user"); user == "" && os.Getuid() == 0 {
		log.Fatal("Please specify user that will run gitlab-runner service")
	}

	if configFile := c.String("config"); configFile != "" {
		// try to load existing config
		config := common.NewConfig()
		err := config.LoadConfig(configFile)
		if err != nil {
			return err
		}

		// save config for the first time
		if !config.Loaded {
			err = config.SaveConfig(configFile)
			if err != nil {
				return err
			}
		}
	}
	return service.Control(s, "install")
}

func RunServiceControl(c *cli.Context) {
	// detect whether we want to install as user service or system service
	isUserService := os.Getuid() != 0
	if runtime.GOOS == "windows" {
		isUserService = true
	}

	// when installing service as system wide service don't specify username for service
	serviceUserName := c.String("user")
	if !isUserService {
		serviceUserName = ""
	}

	if isUserService && runtime.GOOS == "linux" {
		log.Fatal("Please run the commands as root")
	}

	svcConfig := &service.Config{
		Name:        c.String("service"),
		DisplayName: c.String("service"),
		Description: defaultDescription,
		Arguments:   []string{"run"},
		UserName:    serviceUserName,
	}

	switch runtime.GOOS {
	case "darwin":
		svcConfig.Option = service.KeyValue{
			"KeepAlive":     true,
			"RunAtLoad":     true,
			"SessionCreate": true,
			"UserService":   isUserService,
		}

	case "windows":
		svcConfig.Option = service.KeyValue{
			"Password": c.String("password"),
		}
	}

	if wd := c.String("working-directory"); wd != "" {
		svcConfig.Arguments = append(svcConfig.Arguments, "--working-directory", wd)
	}

	if config := c.String("config"); config != "" {
		svcConfig.Arguments = append(svcConfig.Arguments, "--config", config)
	}

	if sn := c.String("service"); sn != "" {
		svcConfig.Arguments = append(svcConfig.Arguments, "--service", sn)
	}

	if user := c.String("user"); !isUserService && user != "" {
		svcConfig.Arguments = append(svcConfig.Arguments, "--user", user)
	}

	s, err := service.New(&NullService{}, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	switch c.Command.Name {
	case "install":
		err = runServiceInstall(s, c)
	default:
		err = service.Control(s, c.Command.Name)
	}

	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	flags := []cli.Flag{
		cli.StringFlag{
			Name:  "service, n",
			Value: defaultServiceName,
			Usage: "Specify service name to use",
		},
	}

	installFlags := flags
	installFlags = append(installFlags, cli.StringFlag{
		Name:  "working-directory, d",
		Value: helpers.GetCurrentWorkingDirectory(),
		Usage: "Specify custom root directory where all data are stored",
	})
	installFlags = append(installFlags, cli.StringFlag{
		Name:  "config, c",
		Value: getDefaultConfigFile(),
		Usage: "Specify custom config file",
	})

	if runtime.GOOS == "windows" {
		installFlags = append(installFlags, cli.StringFlag{
			Name:  "user, u",
			Value: helpers.GetCurrentUserName(),
			Usage: "Specify user-name to secure the runner",
		})
		installFlags = append(installFlags, cli.StringFlag{
			Name:  "password, p",
			Value: "",
			Usage: "Specify user password to install service (required)",
		})
	} else if os.Getuid() == 0 {
		installFlags = append(installFlags, cli.StringFlag{
			Name:  "user, u",
			Value: "",
			Usage: "Specify user-name to secure the runner",
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
