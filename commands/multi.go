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
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/service"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/network"
	"math"
)

type RunnerHealth struct {
	failures  int
	lastCheck time.Time
}

type RunCommand struct {
	configOptions
	network common.Network

	ServiceName      string `short:"n" long:"service" description:"Use different names for different services"`
	WorkingDirectory string `short:"d" long:"working-directory" description:"Specify custom working directory"`
	User             string `short:"u" long:"user" description:"Use specific user to execute shell scripts"`
	Syslog           bool   `long:"syslog" description:"Log to syslog"`

	builds          []*common.Build
	buildsLock      sync.RWMutex
	healthy         map[string]*RunnerHealth
	healthyLock     sync.Mutex
	finished        bool
	abortBuilds     chan os.Signal
	interruptSignal chan os.Signal
	reloadSignal    chan os.Signal
	doneSignal      chan int
}

func (mr *RunCommand) log() *log.Entry {
	return log.WithField("builds", len(mr.builds))
}

func (mr *RunCommand) getHealth(runner *common.RunnerConfig) *RunnerHealth {
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

func (mr *RunCommand) isHealthy(runner *common.RunnerConfig) bool {
	health := mr.getHealth(runner)
	if health.failures < common.HealthyChecks {
		return true
	}

	if time.Since(health.lastCheck) > common.HealthCheckInterval*time.Second {
		mr.log().Errorln("Runner", runner.ShortDescription(), "is not healthy, but will be checked!")
		health.failures = 0
		health.lastCheck = time.Now()
		return true
	}

	return false
}

func (mr *RunCommand) makeHealthy(runner *common.RunnerConfig) {
	health := mr.getHealth(runner)
	health.failures = 0
	health.lastCheck = time.Now()
}

func (mr *RunCommand) makeUnhealthy(runner *common.RunnerConfig) {
	health := mr.getHealth(runner)
	health.failures++

	if health.failures >= common.HealthyChecks {
		mr.log().Errorln("Runner", runner.ShortDescription(), "is not healthy and will be disabled!")
	}
}

func (mr *RunCommand) addBuild(newBuild *common.Build) {
	mr.buildsLock.Lock()
	defer mr.buildsLock.Unlock()

	newBuild.AssignID(mr.builds...)
	mr.builds = append(mr.builds, newBuild)
	mr.log().Debugln("Added a new build", newBuild)
}

func (mr *RunCommand) removeBuild(deleteBuild *common.Build) bool {
	mr.buildsLock.Lock()
	defer mr.buildsLock.Unlock()

	for idx, build := range mr.builds {
		if build == deleteBuild {
			mr.builds = append(mr.builds[0:idx], mr.builds[idx+1:]...)
			mr.log().Debugln("Build removed", deleteBuild)
			return true
		}
	}
	return false
}

func (mr *RunCommand) buildsForRunner(runner *common.RunnerConfig) int {
	count := 0
	for _, build := range mr.builds {
		if build.Runner == runner {
			count++
		}
	}
	return count
}

func (mr *RunCommand) requestBuild(runner *common.RunnerConfig) *common.Build {
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

	buildData, healthy := mr.network.GetBuild(*runner)
	if healthy {
		mr.makeHealthy(runner)
	} else {
		mr.makeUnhealthy(runner)
	}

	if buildData == nil {
		return nil
	}

	mr.log().Debugln("Received new build for", runner.ShortDescription(), "build", buildData.ID)
	newBuild := &common.Build{
		GetBuildResponse: *buildData,
		Runner:           runner,
		BuildAbort:       mr.abortBuilds,
		Network:          mr.network,
	}
	return newBuild
}

func (mr *RunCommand) feedRunners(runners chan *common.RunnerConfig) {
	for !mr.finished {
		mr.log().Debugln("Feeding runners to channel")
		config := mr.config
		for _, runner := range config.Runners {
			runners <- runner
		}
		time.Sleep(common.CheckInterval * time.Second)
	}
}

func (mr *RunCommand) processRunners(id int, stopWorker chan bool, runners chan *common.RunnerConfig) {
	mr.log().Debugln("Starting worker", id)
	for !mr.finished {
		select {
		case runner := <-runners:
			mr.log().Debugln("Checking runner", runner, "on", id)
			newJob := mr.requestBuild(runner)
			if newJob == nil {
				break
			}

			mr.addBuild(newJob)
			newJob.Run(mr.config)
			mr.removeBuild(newJob)
			newJob = nil

			// force GC cycle after processing build
			runtime.GC()

		case <-stopWorker:
			mr.log().Debugln("Stopping worker", id)
			return
		}
	}
	<-stopWorker
}

func (mr *RunCommand) startWorkers(startWorker chan int, stopWorker chan bool, runners chan *common.RunnerConfig) {
	for !mr.finished {
		id := <-startWorker
		go mr.processRunners(id, stopWorker, runners)
	}
}

func (mr *RunCommand) loadConfig() error {
	err := mr.configOptions.loadConfig()
	if err != nil {
		return err
	}

	// pass user to execute scripts as specific user
	if mr.User != "" {
		mr.config.User = &mr.User
	}

	mr.healthy = nil
	return nil
}

func (mr *RunCommand) Start(s service.Service) error {
	mr.builds = []*common.Build{}
	mr.abortBuilds = make(chan os.Signal)
	mr.interruptSignal = make(chan os.Signal, 1)
	mr.reloadSignal = make(chan os.Signal, 1)
	mr.doneSignal = make(chan int, 1)
	mr.log().Println("Starting multi-runner from", mr.ConfigFile, "...")

	if len(mr.WorkingDirectory) > 0 {
		err := os.Chdir(mr.WorkingDirectory)
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

func (mr *RunCommand) Run() {
	runners := make(chan *common.RunnerConfig)
	go mr.feedRunners(runners)

	startWorker := make(chan int)
	stopWorker := make(chan bool)
	go mr.startWorkers(startWorker, stopWorker, runners)

	signal.Notify(mr.reloadSignal, syscall.SIGHUP)
	signal.Notify(mr.interruptSignal, syscall.SIGQUIT)

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
			info, err := os.Stat(mr.ConfigFile)
			if err != nil {
				mr.log().Errorln("Failed to stat config", err)
				break
			}

			if !mr.config.ModTime.Before(info.ModTime()) {
				break
			}

			err = mr.loadConfig()
			if err != nil {
				mr.log().Errorln("Failed to load config", err)
				// don't reload the same file
				mr.config.ModTime = info.ModTime()
				break
			}

			mr.log().Println("Config reloaded.")

		case <-mr.reloadSignal:
			err := mr.loadConfig()
			if err != nil {
				mr.log().Errorln("Failed to load config", err)
				break
			}

			mr.log().Println("Config reloaded.")

		case signaled = <-mr.interruptSignal:
			break finish_worker
		}
	}
	mr.finished = true

	// Pump signal to abort all builds
	go func() {
		for signaled == syscall.SIGQUIT {
			mr.log().Warningln("Requested quit, waiting for builds to finish")
			signaled = <-mr.interruptSignal
		}
		for {
			mr.abortBuilds <- signaled
		}
	}()

	// Wait for workers to shutdown
	for currentWorkers > 0 {
		stopWorker <- true
		currentWorkers--
	}
	mr.log().Println("All workers stopped. Can exit now")
	mr.doneSignal <- 0
}

func (mr *RunCommand) Stop(s service.Service) error {
	mr.log().Warningln("Requested service stop")
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

func (c *RunCommand) Execute(context *cli.Context) {
	svcConfig := &service.Config{
		Name:        c.ServiceName,
		DisplayName: c.ServiceName,
		Description: defaultDescription,
		Arguments:   []string{"run"},
	}

	service, err := service_helpers.New(c, svcConfig)
	if err != nil {
		log.Fatalln(err)
	}

	if c.Syslog {
		logger, err := service.SystemLogger(nil)
		if err == nil {
			log.AddHook(&ServiceLogHook{logger})
		} else {
			log.Errorln(err)
		}
	}

	err = service.Run()
	if err != nil {
		log.Fatalln(err)
	}
}

func init() {
	common.RegisterCommand2("run", "run multi runner service", &RunCommand{
		ServiceName: defaultServiceName,
		network:     &network.GitLabClient{},
	})
}
