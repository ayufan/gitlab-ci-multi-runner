package commands

import (
	"bufio"
	"os"
	"os/signal"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/ssh"
)

type RegisterCommand struct {
	context           *cli.Context
	reader            *bufio.Reader
	registered        bool

	configOptions
	TagList           string              `long:"tag-list" env:"RUNNER_TAG_LIST" description:"Tag list"`
	NonInteractive    bool                `short:"n" long:"non-interactive" env:"REGISTER_NON_INTERACTIVE" description:"Run registration unattended"`
	LeaveRunner       bool                `long:"leave-runner" env:"REGISTER_LEAVE_RUNNER" description:"Don't remove runner if registration fails"`
	RegistrationToken string              `short:"r" long:"registration-token" env:"REGISTRATION_TOKEN" description:"Runner's registration token"`

	common.RunnerConfig
	DockerMySQL       string              `long:"docker-mysql" env:"DOCKER_MYSQL" description:"MySQL version (or specify latest) to link as service Docker service"`
	DockerPostgreSQL  string              `long:"docker-postgres" env:"DOCKER_POSTGRES" description:"PostgreSQL version (or specify latest) to link as service Docker service"`
	DockerMongoDB     string              `long:"docker-mongo" env:"DOCKER_MONGO" description:"MongoDB version (or specify latest) to link as service Docker service"`
	DockerRedis       string              `long:"docker-redis" env:"DOCKER_REDIS" description:"Redis version (or specify latest) to link as service Docker service"`
}

func (s *RegisterCommand) ask(key, prompt string, allowEmptyOptional ...bool) string {
	allowEmpty := len(allowEmptyOptional) > 0 && allowEmptyOptional[0]

	result := s.context.String(key)
	result = strings.TrimSpace(result)

	if s.NonInteractive || prompt == "" {
		if result == "" && !allowEmpty {
			log.Fatalln("The", key, "needs to be entered")
		}
		return result
	}

	for {
		println(prompt)
		if result != "" {
			print("["+result, "]: ")
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
			return newResult
		}

		if allowEmpty || result != "" {
			return result
		}
	}
}

func (s *RegisterCommand) askExecutor() {
	for {
		names := common.GetExecutors()
		executors := strings.Join(names, ", ")
		s.Executor = s.ask("executor", "Please enter the executor: "+executors+":", true)
		if common.NewExecutor(s.Executor) != nil {
			return
		} else {
			message := "Invalid executor specified"
			if s.NonInteractive {
				log.Fatalln(message)
			} else {
				log.Errorln(message)
			}
		}
	}
}

func (s *RegisterCommand) askForDockerService(service string) bool {
	for {
		result := s.ask("docker-"+service, "If you want to enable "+service+" please enter version (X.Y) or enter latest?", true)
		if len(result) == 0 {
			return false
		}
		if result != "latest" {
			_, err := strconv.ParseFloat(result, 32)
			if err != nil {
				println("Invalid version specified", err)
				continue
			}
		}
		s.Docker.Services = append(s.Docker.Services, service+":"+result)
		return true
	}
}

func (s *RegisterCommand) askDocker() {
	if s.Docker == nil {
		s.Docker = &common.DockerConfig{}
	}
	s.Docker.Image = s.ask("docker-image", "Please enter the Docker image (eg. ruby:2.1):")

	if s.askForDockerService("mysql") {
		s.Environment = append(s.Environment, "MYSQL_ALLOW_EMPTY_PASSWORD=1")
	}

	s.askForDockerService("postgres")
	s.askForDockerService("redis")
	s.askForDockerService("mongo")

	s.Docker.Volumes = append(s.Docker.Volumes, "/cache")
}

func (s *RegisterCommand) askParallels() {
	s.Parallels.BaseName = s.ask("parallels-vm", "Please enter the Parallels VM (eg. my-vm):")
}

func (s *RegisterCommand) askSSHServer() {
	if host := s.ask("ssh-host", "Please enter the SSH server address (eg. my.server.com):"); host != "" {
		s.SSH.Host = &host
	}
	if port := s.ask("ssh-port", "Please enter the SSH server port (eg. 22):", true); port != "" {
		s.SSH.Port = &port
	}
}

func (s *RegisterCommand) askSSHLogin() {
	if user := s.ask("ssh-user", "Please enter the SSH user (eg. root):"); user != "" {
		s.SSH.User = &user
	}
	if password := s.ask("ssh-password", "Please enter the SSH password (eg. docker.io):", true); password != "" {
		s.SSH.Password = &password
	}
	if identityFile := s.ask("ssh-identity-file", "Please enter path to SSH identity file (eg. /home/user/.ssh/id_rsa):", true); identityFile != "" {
		s.SSH.IdentityFile = &identityFile
	}
}

func (s *RegisterCommand) addRunner(runner *common.RunnerConfig) {
	s.config.Runners = append(s.config.Runners, runner)
}

func (s *RegisterCommand) askRunner() {
	s.URL = s.ask("url", "Please enter the gitlab-ci coordinator URL (e.g. https://gitlab.com/ci):")

	if s.Token != "" {
		log.Infoln("Token specified trying to verify runner...")
		log.Warningln("If you want to register use the '-r' instead of '-t'.")
		if !common.VerifyRunner(s.URL, s.Token) {
			log.Fatalln("Failed to verify this runner. Perhaps you are having network problems")
		}
	} else {
		s.RegistrationToken = s.ask("registration-token", "Please enter the gitlab-ci token for this runner:")
		s.Name = s.ask("name", "Please enter the gitlab-ci description for this runner:")
		s.TagList = s.ask("tag-list", "Please enter the gitlab-ci tags for this runner (comma separated):", true)

		result := common.RegisterRunner(s.URL, s.RegistrationToken, s.Name, s.TagList)
		if result == nil {
			log.Fatalln("Failed to register this runner. Perhaps you are having network problems")
		}
		
		s.Token = result.Token
		s.registered = true
	}
}

func (c *RegisterCommand) Execute(context *cli.Context) {
	c.context = context
	err := c.loadConfig()
	if err != nil {
		log.Fatalln(err)
	}
	c.askRunner()

	if !c.LeaveRunner {
		defer func() {
			if r := recover(); r != nil {
				if c.registered {
					common.DeleteRunner(c.URL, c.Token)
				}

				// pass panic to next defer
				panic(r)
			}
		}()

		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)

		go func() {
			s := <-signals
			common.DeleteRunner(c.URL, c.Token)
			log.Fatalf("RECEIVED SIGNAL: %v", s)
		}()
	}

	c.askExecutor()

	if limit := helpers.NonZeroOrDefault(c.Limit, 0); c.config.Concurrent < limit {
		log.Warningf("Specified limit (%d) larger then current concurrent limit (%d). Concurrent limit will not be enlarged.", limit, c.config.Concurrent)
	}

	switch c.Executor {
	case "docker":
		c.askDocker()
		c.SSH = nil
		c.Parallels = nil
	case "docker-ssh":
		c.askDocker()
		c.askSSHLogin()
		c.Parallels = nil
	case "ssh":
		c.askSSHServer()
		c.askSSHLogin()
		c.Docker = nil
		c.Parallels = nil
	case "parallels":
		c.askParallels()
		c.askSSHServer()
		c.Docker = nil
	}

	c.addRunner(&c.RunnerConfig)
	c.saveConfig()

	log.Printf("Runner registered successfully. Feel free to start it, but if it's running already the config should be automatically reloaded!")
}

func getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

func init() {
	common.RegisterCommand2("register", "register a new runner", &RegisterCommand{
		RunnerConfig: common.RunnerConfig{
			Name:      getHostname(),
			Parallels: &common.ParallelsConfig{},
			SSH:       &ssh.Config{},
			Docker:    &common.DockerConfig{},
		},
	})
}
