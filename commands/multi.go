package commands

import (
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	service "github.com/ayufan/golang-kardianos-service"
	"github.com/codegangsta/cli"

	log "github.com/Sirupsen/logrus"

	"errors"
	"fmt"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"math"
)

type RunnerHealth struct {
	failures  int
	lastCheck time.Time
}

type MultiRunner struct {
	config           *common.Config
	configFile       string
	workingDirectory string
	builds           []*common.Build
	buildsLock       sync.RWMutex
	healthy          map[string]*RunnerHealth
	healthyLock      sync.Mutex
	finished         bool
	abortBuilds      chan os.Signal
	interruptSignal  chan os.Signal
	reloadSignal     chan os.Signal
	doneSignal       chan int
}

func (mr *MultiRunner) errorln(args ...interface{}) {
	args = append([]interface{}{len(mr.builds)}, args...)
	log.Errorln(args...)
}

func (mr *MultiRunner) warningln(args ...interface{}) {
	args = append([]interface{}{len(mr.builds)}, args...)
	log.Warningln(args...)
}

func (mr *MultiRunner) debugln(args ...interface{}) {
	args = append([]interface{}{len(mr.builds)}, args...)
	log.Debugln(args...)
}

func (mr *MultiRunner) println(args ...interface{}) {
	args = append([]interface{}{len(mr.builds)}, args...)
	log.Println(args...)
}

func (mr *MultiRunner) getHealth(runner *common.RunnerConfig) *RunnerHealth {
	mr.healthyLock.Lock()
	defer mr.healthyLock.Unlock()

	if mr.healthy == nil {
		mr.healthy = map[string]*RunnerHealth{}
	}
	health := mr.healthy[runner.UniqueID()]
	if health == nil {
		health = &RunnerHealth{
			lastCheck: time.Now(),
		}
		mr.healthy[runner.UniqueID()] = health
	}
	return health
}

func (mr *MultiRunner) isHealthy(runner *common.RunnerConfig) bool {
	health := mr.getHealth(runner)
	if health.failures < common.HealthyChecks {
		return true
	}

	if time.Since(health.lastCheck) > common.HealthCheckInterval*time.Second {
		mr.errorln("Runner", runner.ShortDescription(), "is not healthy, but will be checked!")
		health.failures = 0
		health.lastCheck = time.Now()
		return true
	}

	return false
}

func (mr *MultiRunner) makeHealthy(runner *common.RunnerConfig) {
	health := mr.getHealth(runner)
	health.failures = 0
	health.lastCheck = time.Now()
}

func (mr *MultiRunner) makeUnhealthy(runner *common.RunnerConfig) {
	health := mr.getHealth(runner)
	health.failures++

	if health.failures >= common.HealthyChecks {
		mr.errorln("Runner", runner.ShortDescription(), "is not healthy and will be disabled!")
	}
}

func (mr *MultiRunner) addBuild(newBuild *common.Build) {
	mr.buildsLock.Lock()
	defer mr.buildsLock.Unlock()

	newBuild.AssignID(mr.builds...)
	mr.builds = append(mr.builds, newBuild)
	mr.debugln("Added a new build", newBuild)
}

func (mr *MultiRunner) removeBuild(deleteBuild *common.Build) bool {
	mr.buildsLock.Lock()
	defer mr.buildsLock.Unlock()

	for idx, build := range mr.builds {
		if build == deleteBuild {
			mr.builds = append(mr.builds[0:idx], mr.builds[idx+1:]...)
			mr.debugln("Build removed", deleteBuild)
			return true
		}
	}
	return false
}

func (mr *MultiRunner) buildsForRunner(runner *common.RunnerConfig) int {
	count := 0
	for _, build := range mr.builds {
		if build.Runner == runner {
			count++
		}
	}
	return count
}

func (mr *MultiRunner) requestBuild(runner *common.RunnerConfig) *common.Build {
	if runner == nil {
		return nil
	}

	if !mr.isHealthy(runner) {
		return nil
	}

	count := mr.buildsForRunner(runner)
	limit := helpers.NonZeroOrDefault(runner.Limit, math.MaxInt32)
	if count >= limit {
		return nil
	}

	buildData, healthy := common.GetBuild(*runner)
	if healthy {
		mr.makeHealthy(runner)
	} else {
		mr.makeUnhealthy(runner)
	}

	if buildData == nil {
		return nil
	}

	mr.debugln("Received new build for", runner.ShortDescription(), "build", buildData.ID)
	newBuild := &common.Build{
		GetBuildResponse: *buildData,
		Runner:           runner,
		BuildAbort:       mr.abortBuilds,
	}
	return newBuild
}

func (mr *MultiRunner) feedRunners(runners chan *common.RunnerConfig) {
	for !mr.finished {
		mr.debugln("Feeding runners to channel")
		config := mr.config
		for _, runner := range config.Runners {
			runners <- runner
		}
		time.Sleep(common.CheckInterval * time.Second)
	}
}

func (mr *MultiRunner) processRunners(id int, stopWorker chan bool, runners chan *common.RunnerConfig) {
	mr.debugln("Starting worker", id)
	for !mr.finished {
		select {
		case runner := <-runners:
			mr.debugln("Checking runner", runner, "on", id)
			newJob := mr.requestBuild(runner)
			if newJob == nil {
				break
			}

			mr.addBuild(newJob)
			newJob.Run()
			mr.removeBuild(newJob)
			newJob = nil

			// force GC cycle after processing build
			runtime.GC()

		case <-stopWorker:
			mr.debugln("Stopping worker", id)
			return
		}
	}
	<-stopWorker
}

func (mr *MultiRunner) startWorkers(startWorker chan int, stopWorker chan bool, runners chan *common.RunnerConfig) {
	for !mr.finished {
		id := <-startWorker
		go mr.processRunners(id, stopWorker, runners)
	}
}

func (mr *MultiRunner) loadConfig() error {
	newConfig := common.NewConfig()
	err := newConfig.LoadConfig(mr.configFile)
	if err != nil {
		return err
	}

	mr.healthy = nil
	mr.config = newConfig
	return nil
}

func (mr *MultiRunner) Start(s service.Service) error {
	mr.builds = []*common.Build{}
	mr.abortBuilds = make(chan os.Signal)
	mr.interruptSignal = make(chan os.Signal)
	mr.reloadSignal = make(chan os.Signal, 1)
	mr.doneSignal = make(chan int)

	mr.println("Starting multi-runner from", mr.configFile, "...")

	if len(mr.workingDirectory) > 0 {
		err := os.Chdir(mr.workingDirectory)
		if err != nil {
			return err
		}
	}

	err := mr.loadConfig()
	if err != nil {
		panic(err)
	}

	// Start should not block. Do the actual work async.
	go mr.Run()

	return nil
}

func (mr *MultiRunner) Run() {
	runners := make(chan *common.RunnerConfig)
	go mr.feedRunners(runners)

	startWorker := make(chan int)
	stopWorker := make(chan bool)
	go mr.startWorkers(startWorker, stopWorker, runners)

	signal.Notify(mr.reloadSignal, syscall.SIGHUP)

	currentWorkers := 0
	workerIndex := 0

	var signaled os.Signal

finish_worker:
	for {
		buildLimit := mr.config.Concurrent

		for currentWorkers > buildLimit {
			select {
			case stopWorker <- true:
			case signaled = <-mr.interruptSignal:
				break finish_worker
			}
			currentWorkers--
		}

		for currentWorkers < buildLimit {
			select {
			case startWorker <- workerIndex:
			case signaled = <-mr.interruptSignal:
				break finish_worker
			}
			currentWorkers++
			workerIndex++
		}

		select {
		case <-time.After(common.ReloadConfigInterval * time.Second):
			info, err := os.Stat(mr.configFile)
			if err != nil {
				mr.errorln("Failed to stat config", err)
				break
			}

			if !mr.config.ModTime.Before(info.ModTime()) {
				break
			}

			err = mr.loadConfig()
			if err != nil {
				mr.errorln("Failed to load config", err)
				// don't reload the same file
				mr.config.ModTime = info.ModTime()
				break
			}

			mr.println("Config reloaded.")

		case <-mr.reloadSignal:
			err := mr.loadConfig()
			if err != nil {
				mr.errorln("Failed to load config", err)
				break
			}

			mr.println("Config reloaded.")

		case signaled = <-mr.interruptSignal:
			break finish_worker
		}
	}
	mr.finished = true

	// Pump signal to abort all builds
	go func() {
		for {
			mr.abortBuilds <- signaled
		}
	}()

	// Wait for workers to shutdown
	for currentWorkers > 0 {
		stopWorker <- true
		currentWorkers--
	}
	mr.println("All workers stopped. Can exit now")
	mr.doneSignal <- 0
}

func (mr *MultiRunner) Stop(s service.Service) error {
	mr.warningln("Requested service stop")
	mr.interruptSignal <- os.Interrupt

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	select {
	case newSignal := <-signals:
		return fmt.Errorf("forced exit: %v", newSignal)
	case <-time.After(common.ShutdownTimeout * time.Second):
		return errors.New("shutdown timedout")
	case <-mr.doneSignal:
		return nil
	}
}

func CreateService(c *cli.Context) service.Service {
	serviceName := c.String("service-name")
	displayName := c.String("service-name")
	if serviceName == "" {
		serviceName = defaultServiceName
		displayName = defaultDisplayName
	}

	svcConfig := &service.Config{
		Name:        serviceName,
		DisplayName: displayName,
		Description: defaultDescription,
		Arguments:   []string{"run"},
	}

	mr := &MultiRunner{
		configFile:       c.String("config"),
		workingDirectory: c.String("working-directory"),
	}

	s, err := service.New(mr, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	return s
}

func RunService(c *cli.Context) {
	s := CreateService(c)

	if c.Bool("syslog") {
		logger, err := s.Logger(nil)
		if err != nil {
			log.Fatal(err)
		}

		log.AddHook(&ServiceLogHook{logger})
	}

	err := s.Run()
	if err != nil {
		log.Errorln(err)
	}
}

func init() {
	common.RegisterCommand(cli.Command{
		Name:      "run",
		ShortName: "r",
		Usage:     "run multi runner service",
		Action:    RunService,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "docker-host",
				Value:  "",
				Usage:  "Docker endpoint URL",
				EnvVar: "DOCKER_HOST",
			},
			cli.StringFlag{
				Name:   "config",
				Value:  getDefaultConfigFile(),
				Usage:  "Config file",
				EnvVar: "CONFIG_FILE",
			},
			cli.StringFlag{
				Name:  "working-directory, d",
				Usage: "Specify custom working directory",
			},
			cli.StringFlag{
				Name:  "service-name, n",
				Value: "",
				Usage: "Use different names for different services",
			},
			cli.BoolFlag{
				Name:  "syslog",
				Usage: "Log to syslog",
			},
		},
	})
}
