package commands

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/ayufan/golang-kardianos-service"
	"github.com/codegangsta/cli"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/service"
	"os"
	"runtime"
)

const (
	defaultServiceName = "gitlab-runner"
	defaultDisplayName = "GitLab Runner"
	defaultDescription = "GitLab Runner"
)

type ServiceLogHook struct {
	service.Logger
}

func (s *ServiceLogHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	}
}

func (s *ServiceLogHook) Fire(e *logrus.Entry) error {
	switch e.Level {
	case logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel:
		s.Error(e.Message)
	case logrus.WarnLevel:
		s.Warning(e.Message)
	case logrus.InfoLevel:
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
		logrus.Fatal("Please specify user that will run gitlab-runner service")
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

func runServiceStatus(displayName string, s service.Service, c *cli.Context) error {
	err := s.Status()
	if err == nil {
		fmt.Println(displayName+":", "Service is running!")
	} else {
		fmt.Fprintln(os.Stderr, displayName+":", err)
		os.Exit(1)
	}
	return nil
}

func getServiceArguments(c *cli.Context, isUserService bool) (arguments []string) {
	if wd := c.String("working-directory"); wd != "" {
		arguments = append(arguments, "--working-directory", wd)
	}

	if config := c.String("config"); config != "" {
		arguments = append(arguments, "--config", config)
	}

	if sn := c.String("service"); sn != "" {
		arguments = append(arguments, "--service", sn)
	}

	if user := c.String("user"); !isUserService && user != "" {
		arguments = append(arguments, "--user", user)
	}

	arguments = append(arguments, "--syslog")
	return
}

func createServiceConfig(c *cli.Context) (svcConfig *service.Config) {
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
		logrus.Fatal("Please run the commands as root")
	}

	svcConfig = &service.Config{
		Name:        c.String("service"),
		DisplayName: c.String("service"),
		Description: defaultDescription,
		Arguments:   []string{"run"},
		UserName:    serviceUserName,
	}
	svcConfig.Arguments = append(svcConfig.Arguments, getServiceArguments(c, isUserService)...)

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
	return
}

func RunServiceControl(c *cli.Context) {
	svcConfig := createServiceConfig(c)

	s, err := service_helpers.New(&NullService{}, svcConfig)
	if err != nil {
		logrus.Fatal(err)
	}

	switch c.Command.Name {
	case "install":
		err = runServiceInstall(s, c)
	case "status":
		err = runServiceStatus(svcConfig.DisplayName, s, c)
	default:
		err = service.Control(s, c.Command.Name)
	}

	if err != nil {
		logrus.Fatal(err)
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
			Value: "",
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
	common.RegisterCommand(cli.Command{
		Name:   "status",
		Usage:  "get status of a service",
		Action: RunServiceControl,
		Flags:  flags,
	})
}
