package commands

import (
	"bufio"
	"os"
	"os/signal"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/ssh"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/network"
)

type RegisterCommand struct {
	context    *cli.Context
	network    common.Network
	reader     *bufio.Reader
	registered bool

	configOptions
	TagList           string `long:"tag-list" env:"RUNNER_TAG_LIST" description:"Tag list"`
	NonInteractive    bool   `short:"n" long:"non-interactive" env:"REGISTER_NON_INTERACTIVE" description:"Run registration unattended"`
	LeaveRunner       bool   `long:"leave-runner" env:"REGISTER_LEAVE_RUNNER" description:"Don't remove runner if registration fails"`
	RegistrationToken string `short:"r" long:"registration-token" env:"REGISTRATION_TOKEN" description:"Runner's registration token"`

	common.RunnerConfig
}

func (s *RegisterCommand) askOnce(prompt string, result *string, allowEmpty bool) bool {
	println(prompt)
	if *result != "" {
		print("["+*result, "]: ")
	}

	if s.reader == nil {
		s.reader = bufio.NewReader(os.Stdin)
	}

	data, _, err := s.reader.ReadLine()
	if err != nil {
		panic(err)
	}
	newResult := string(data)
	newResult = strings.TrimSpace(newResult)

	if newResult != "" {
		*result = newResult
		return true
	}

	if allowEmpty || *result != "" {
		return true
	}
	return false
}

func (s *RegisterCommand) ask(key, prompt string, allowEmptyOptional ...bool) string {
	allowEmpty := len(allowEmptyOptional) > 0 && allowEmptyOptional[0]

	result := s.context.String(key)
	result = strings.TrimSpace(result)

	if s.NonInteractive || prompt == "" {
		if result == "" && !allowEmpty {
			log.Panicln("The", key, "needs to be entered")
		}
		return result
	}

	for {
		if s.askOnce(prompt, &result, allowEmpty) {
			break
		}
	}

	return result
}

func (s *RegisterCommand) askExecutor() {
	for {
		names := common.GetExecutors()
		executors := strings.Join(names, ", ")
		s.Executor = s.ask("executor", "Please enter the executor: "+executors+":", true)
		if common.GetExecutor(s.Executor) != nil {
			return
		}

		message := "Invalid executor specified"
		if s.NonInteractive {
			log.Panicln(message)
		} else {
			log.Errorln(message)
		}
	}
}

func (s *RegisterCommand) askDocker() {
	if s.Docker == nil {
		s.Docker = &common.DockerConfig{}
	}
	s.Docker.Image = s.ask("docker-image", "Please enter the default Docker image (eg. ruby:2.1):")
	s.Docker.Volumes = append(s.Docker.Volumes, "/cache")
}

func (s *RegisterCommand) askParallels() {
	s.Parallels.BaseName = s.ask("parallels-vm", "Please enter the Parallels VM (eg. my-vm):")
}

func (s *RegisterCommand) askVirtualBox() {
	s.VirtualBox.BaseName = s.ask("virtualbox-vm", "Please enter the VirtualBox VM (eg. my-vm):")
}

func (s *RegisterCommand) askSSHServer() {
	s.SSH.Host = s.ask("ssh-host", "Please enter the SSH server address (eg. my.server.com):")
	s.SSH.Port = s.ask("ssh-port", "Please enter the SSH server port (eg. 22):", true)
}

func (s *RegisterCommand) askSSHLogin() {
	s.SSH.User = s.ask("ssh-user", "Please enter the SSH user (eg. root):")
	s.SSH.Password = s.ask("ssh-password", "Please enter the SSH password (eg. docker.io):", true)
	s.SSH.IdentityFile = s.ask("ssh-identity-file", "Please enter path to SSH identity file (eg. /home/user/.ssh/id_rsa):", true)
}

func (s *RegisterCommand) addRunner(runner *common.RunnerConfig) {
	s.config.Runners = append(s.config.Runners, runner)
}

func (s *RegisterCommand) askRunner() {
	s.URL = s.ask("url", "Please enter the gitlab-ci coordinator URL (e.g. https://gitlab.com/):")

	if s.Token != "" {
		log.Infoln("Token specified trying to verify runner...")
		log.Warningln("If you want to register use the '-r' instead of '-t'.")
		if !s.network.VerifyRunner(s.RunnerCredentials) {
			log.Panicln("Failed to verify this runner. Perhaps you are having network problems")
		}
	} else {
		// we store registration token as token, since we pass that to RunnerCredentials
		s.Token = s.ask("registration-token", "Please enter the gitlab-ci token for this runner:")
		s.Name = s.ask("name", "Please enter the gitlab-ci description for this runner:")
		s.TagList = s.ask("tag-list", "Please enter the gitlab-ci tags for this runner (comma separated):", true)

		result := s.network.RegisterRunner(s.RunnerCredentials, s.Name, s.TagList)
		if result == nil {
			log.Panicln("Failed to register this runner. Perhaps you are having network problems")
		}

		s.Token = result.Token
		s.registered = true
	}
}

func (s *RegisterCommand) askExecutorOptions() {
	switch s.Executor {
	case "docker":
		s.askDocker()
		s.SSH = nil
		s.Parallels = nil
		s.VirtualBox = nil
	case "docker-ssh":
		s.askDocker()
		s.askSSHLogin()
		s.Parallels = nil
		s.VirtualBox = nil
	case "ssh":
		s.askSSHServer()
		s.askSSHLogin()
		s.Docker = nil
		s.Parallels = nil
		s.VirtualBox = nil
	case "parallels":
		s.askParallels()
		s.askSSHServer()
		s.Docker = nil
		s.VirtualBox = nil
	case "VirtualBox":
		s.askVirtualBox()
		s.askSSHLogin()
		s.Docker = nil
		s.Parallels = nil
	}
}

func (s *RegisterCommand) Execute(context *cli.Context) {
	userModeWarning(true)

	s.context = context
	err := s.loadConfig()
	if err != nil {
		log.Panicln(err)
	}
	s.askRunner()

	if !s.LeaveRunner {
		defer func() {
			// De-register runner on panic
			if r := recover(); r != nil {
				if s.registered {
					s.network.DeleteRunner(s.RunnerCredentials)
				}

				// pass panic to next defer
				panic(r)
			}
		}()

		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)

		go func() {
			signal := <-signals
			s.network.DeleteRunner(s.RunnerCredentials)
			log.Fatalf("RECEIVED SIGNAL: %v", signal)
		}()
	}

	s.askExecutor()

	if s.config.Concurrent < s.Limit {
		log.Warningf("Specified limit (%d) larger then current concurrent limit (%d). Concurrent limit will not be enlarged.", s.Limit, s.config.Concurrent)
	}
	s.Machine = nil

	s.askExecutorOptions()
	s.addRunner(&s.RunnerConfig)
	s.saveConfig()

	log.Printf("Runner registered successfully. Feel free to start it, but if it's running already the config should be automatically reloaded!")
}

func getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

func init() {
	common.RegisterCommand2("register", "register a new runner", &RegisterCommand{
		RunnerConfig: common.RunnerConfig{
			Name: getHostname(),
			RunnerSettings: common.RunnerSettings{
				Parallels:  &common.ParallelsConfig{},
				SSH:        &ssh.Config{},
				Docker:     &common.DockerConfig{},
				VirtualBox: &common.VirtualBoxConfig{},
			},
		},
		network: &network.GitLabClient{},
	})
}
