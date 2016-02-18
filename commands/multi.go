package commands

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	service "github.com/ayufan/golang-kardianos-service"
	"github.com/codegangsta/cli"

	log "github.com/Sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/service"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/network"
)

type runnerAcquire struct {
	common.RunnerConfig
	provider common.ExecutorProvider
	data     common.ExecutorData
}

func (r *runnerAcquire) Release() {
	r.provider.Release(&r.RunnerConfig, r.data)
}

type RunCommand struct {
	configOptions
	network common.Network
	healthHelper
	buildsHelper

	ServiceName      string `short:"n" long:"service" description:"Use different names for different services"`
	WorkingDirectory string `short:"d" long:"working-directory" description:"Specify custom working directory"`
	User             string `short:"u" long:"user" description:"Use specific user to execute shell scripts"`
	Syslog           bool   `long:"syslog" description:"Log to syslog"`

	finished        bool
	abortBuilds     chan os.Signal
	interruptSignal chan os.Signal
	reloadSignal    chan os.Signal
	doneSignal      chan int
}

func (mr *RunCommand) log() *log.Entry {
	return log.WithField("builds", len(mr.builds))
}

func (mr *RunCommand) feedRunner(runner *common.RunnerConfig, runners chan *runnerAcquire) {
	if !mr.isHealthy(runner.UniqueID()) {
		return
	}

	provider := common.GetExecutor(runner.Executor)
	if provider == nil {
		return
	}

	data, err := provider.Acquire(runner)
	if err != nil {
		log.Warningln("Failed to update executor", runner.Executor, "for", runner.ShortDescription(), err)
		return
	}

	runners <- &runnerAcquire{*runner, provider, data}
}

func (mr *RunCommand) feedRunners(runners chan *runnerAcquire) {
	for !mr.finished {
		mr.log().Debugln("Feeding runners to channel")
		config := mr.config
		for _, runner := range config.Runners {
			mr.feedRunner(runner, runners)
		}
		time.Sleep(common.CheckInterval * time.Second)
	}
}

func (mr *RunCommand) processRunner(id int, runner *runnerAcquire) (err error) {
	defer runner.Release()

	// Acquire build slot
	build := mr.buildsHelper.acquire(runner)
	if build == nil {
		return
	}
	defer mr.buildsHelper.release(build)

	// Receive a new build
	buildData, healthy := mr.network.GetBuild(runner.RunnerConfig)
	mr.makeHealthy(runner.UniqueID(), healthy)
	if buildData == nil {
		return
	}

	// Make sure to always close output
	trace := mr.network.ProcessBuild(runner.RunnerConfig, buildData.ID)
	defer trace.Fail(err)

	// Process a build
	build.GetBuildResponse = *buildData
	build.BuildAbort = mr.abortBuilds
	build.Network = mr.network
	err = build.Run(mr.config, trace)
	return
}

func (mr *RunCommand) processRunners(id int, stopWorker chan bool, runners chan *runnerAcquire) {
	mr.log().Debugln("Starting worker", id)
	for !mr.finished {
		select {
		case runner := <-runners:
			mr.processRunner(id, runner)

			// force GC cycle after processing build
			runtime.GC()

		case <-stopWorker:
			mr.log().Debugln("Stopping worker", id)
			return
		}
	}
	<-stopWorker
}

func (mr *RunCommand) startWorkers(startWorker chan int, stopWorker chan bool, runners chan *runnerAcquire) {
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
		mr.config.User = mr.User
	}

	mr.healthy = nil
	mr.log().Println("Config loaded:", helpers.ToYAML(mr.config))
	return nil
}

func (mr *RunCommand) checkConfig() (err error) {
	info, err := os.Stat(mr.ConfigFile)
	if err != nil {
		return err
	}

	if !mr.config.ModTime.Before(info.ModTime()) {
		return nil
	}

	err = mr.loadConfig()
	if err != nil {
		mr.log().Errorln("Failed to load config", err)
		// don't reload the same file
		mr.config.ModTime = info.ModTime()
		return
	}
	return nil
}

func (mr *RunCommand) Start(s service.Service) error {
	mr.builds = []*common.Build{}
	mr.abortBuilds = make(chan os.Signal)
	mr.interruptSignal = make(chan os.Signal, 1)
	mr.reloadSignal = make(chan os.Signal, 1)
	mr.doneSignal = make(chan int, 1)
	mr.log().Println("Starting multi-runner from", mr.ConfigFile, "...")

	userModeWarning(false)

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

func (mr *RunCommand) updateWorkers(currentWorkers, workerIndex *int, startWorker chan int, stopWorker chan bool) os.Signal {
	buildLimit := mr.config.Concurrent

	for *currentWorkers > buildLimit {
		select {
		case stopWorker <- true:
		case signaled := <-mr.interruptSignal:
			return signaled
		}
		*currentWorkers--
	}

	for *currentWorkers < buildLimit {
		select {
		case startWorker <- *workerIndex:
		case signaled := <-mr.interruptSignal:
			return signaled
		}
		*currentWorkers++
		*workerIndex++
	}

	return nil
}

func (mr *RunCommand) updateConfig() os.Signal {
	select {
	case <-time.After(common.ReloadConfigInterval * time.Second):
		err := mr.checkConfig()
		if err != nil {
			mr.log().Errorln("Failed to load config", err)
		}

	case <-mr.reloadSignal:
		err := mr.loadConfig()
		if err != nil {
			mr.log().Errorln("Failed to load config", err)
		}

	case signaled := <-mr.interruptSignal:
		return signaled
	}
	return nil
}

func (mr *RunCommand) Run() {
	runners := make(chan *runnerAcquire)
	go mr.feedRunners(runners)

	startWorker := make(chan int)
	stopWorker := make(chan bool)
	go mr.startWorkers(startWorker, stopWorker, runners)

	signal.Notify(mr.reloadSignal, syscall.SIGHUP)
	signal.Notify(mr.interruptSignal, syscall.SIGQUIT)

	currentWorkers := 0
	workerIndex := 0

	var signaled os.Signal
	for {
		signaled = mr.updateWorkers(&currentWorkers, &workerIndex, startWorker, stopWorker)
		if signaled != nil {
			break
		}

		signaled = mr.updateConfig()
		if signaled != nil {
			break
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

func (mr *RunCommand) Execute(context *cli.Context) {
	svcConfig := &service.Config{
		Name:        mr.ServiceName,
		DisplayName: mr.ServiceName,
		Description: defaultDescription,
		Arguments:   []string{"run"},
	}

	service, err := service_helpers.New(mr, svcConfig)
	if err != nil {
		log.Fatalln(err)
	}

	if mr.Syslog {
		log.SetFormatter(new(log.TextFormatter))
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
