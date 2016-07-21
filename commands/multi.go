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
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/sentry"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers/service"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/network"
)

type RunCommand struct {
	configOptions
	network common.Network
	healthHelper

	buildsHelper buildsHelper

	ServiceName      string `short:"n" long:"service" description:"Use different names for different services"`
	WorkingDirectory string `short:"d" long:"working-directory" description:"Specify custom working directory"`
	User             string `short:"u" long:"user" description:"Use specific user to execute shell scripts"`
	Syslog           bool   `long:"syslog" description:"Log to syslog"`

	sentryLogHook sentry.LogHook

	// abortBuilds is used to abort running builds
	abortBuilds chan os.Signal

	// runSignal is used to abort current operation (scaling workers, waiting for config)
	runSignal chan os.Signal

	// reloadSignal is used to trigger forceful config reload
	reloadSignal chan os.Signal

	// stopSignals is to catch a signals notified to process: SIGTERM, SIGQUIT, Interrupt, Kill
	stopSignals chan os.Signal

	// stopSignal is used to preserve the signal that was used to stop the process
	// In case this is SIGQUIT it makes to finish all buids
	stopSignal os.Signal

	// runFinished is used to notify that Run() did finish
	runFinished chan bool
}

func (mr *RunCommand) log() *log.Entry {
	return log.WithField("builds", len(mr.buildsHelper.builds))
}

func (mr *RunCommand) feedRunner(runner *common.RunnerConfig, runners chan *common.RunnerConfig) {
	if !mr.isHealthy(runner.UniqueID()) {
		return
	}

	runners <- runner
}

func (mr *RunCommand) feedRunners(runners chan *common.RunnerConfig) {
	for mr.stopSignal == nil {
		mr.log().Debugln("Feeding runners to channel")
		config := mr.config

		// If no runners wait full interval to test again
		if len(config.Runners) == 0 {
			time.Sleep(config.GetCheckInterval())
			continue
		}

		interval := config.GetCheckInterval() / time.Duration(len(config.Runners))

		// Feed runner with waiting exact amount of time
		for _, runner := range config.Runners {
			mr.feedRunner(runner, runners)
			time.Sleep(interval)
		}
	}
}

func (mr *RunCommand) processRunner(id int, runner *common.RunnerConfig, runners chan *common.RunnerConfig) (err error) {
	provider := common.GetExecutor(runner.Executor)
	if provider == nil {
		return
	}

	context, err := provider.Acquire(runner)
	if err != nil {
		log.Warningln("Failed to update executor", runner.Executor, "for", runner.ShortDescription(), err)
		return
	}
	defer provider.Release(runner, context)

	// Acquire build slot
	if !mr.buildsHelper.acquire(runner) {
		return
	}
	defer mr.buildsHelper.release(runner)

	// Receive a new build
	buildData, healthy := mr.network.GetBuild(*runner)
	mr.makeHealthy(runner.UniqueID(), healthy)
	if buildData == nil {
		return
	}

	// Make sure to always close output
	buildCredentials := &common.BuildCredentials{
		ID:    buildData.ID,
		Token: buildData.Token,
	}
	trace := mr.network.ProcessBuild(*runner, buildCredentials)
	defer trace.Fail(err)

	// Create a new build
	build := &common.Build{
		GetBuildResponse: *buildData,
		Runner:           runner,
		ExecutorData:     context,
		SystemInterrupt:  mr.abortBuilds,
	}

	// Add build to list of builds to assign numbers
	mr.buildsHelper.addBuild(build)
	defer mr.buildsHelper.removeBuild(build)

	// Process the same runner by different worker again
	// to speed up taking the builds
	select {
	case runners <- runner:
		mr.log().WithField("runner", runner.ShortDescription()).Debugln("Requeued the runner")

	default:
		mr.log().WithField("runner", runner.ShortDescription()).Debugln("Failed to requeue the runner: ")
	}

	// Process a build
	return build.Run(mr.config, trace)
}

func (mr *RunCommand) processRunners(id int, stopWorker chan bool, runners chan *common.RunnerConfig) {
	mr.log().WithField("worker", id).Debugln("Starting worker")
	for mr.stopSignal == nil {
		select {
		case runner := <-runners:
			mr.processRunner(id, runner, runners)

			// force GC cycle after processing build
			runtime.GC()

		case <-stopWorker:
			mr.log().WithField("worker", id).Debugln("Stopping worker")
			return
		}
	}
	<-stopWorker
}

func (mr *RunCommand) startWorkers(startWorker chan int, stopWorker chan bool, runners chan *common.RunnerConfig) {
	for mr.stopSignal == nil {
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
	mr.log().Println("Configuration loaded")
	mr.log().Debugln(helpers.ToYAML(mr.config))

	// initialize sentry
	if mr.config.SentryDSN != nil {
		var err error
		mr.sentryLogHook, err = sentry.NewLogHook(*mr.config.SentryDSN)
		if err != nil {
			mr.log().WithError(err).Errorln("Sentry failure")
		}
	} else {
		mr.sentryLogHook = sentry.LogHook{}
	}

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
	mr.abortBuilds = make(chan os.Signal)
	mr.runSignal = make(chan os.Signal, 1)
	mr.reloadSignal = make(chan os.Signal, 1)
	mr.runFinished = make(chan bool, 1)
	mr.stopSignals = make(chan os.Signal)
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
		return err
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
		case signaled := <-mr.runSignal:
			return signaled
		}
		*currentWorkers--
	}

	for *currentWorkers < buildLimit {
		select {
		case startWorker <- *workerIndex:
		case signaled := <-mr.runSignal:
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

	case signaled := <-mr.runSignal:
		return signaled
	}
	return nil
}

func (mr *RunCommand) runWait() {
	mr.log().Debugln("Waiting for stop signal")

	// Save the stop signal and exit to execute Stop()
	mr.stopSignal = <-mr.stopSignals
}

func (mr *RunCommand) Run() {
	runners := make(chan *common.RunnerConfig)
	go mr.feedRunners(runners)

	signal.Notify(mr.stopSignals, syscall.SIGQUIT, syscall.SIGTERM, os.Interrupt, os.Kill)
	signal.Notify(mr.reloadSignal, syscall.SIGHUP)

	startWorker := make(chan int)
	stopWorker := make(chan bool)
	go mr.startWorkers(startWorker, stopWorker, runners)

	currentWorkers := 0
	workerIndex := 0

	for mr.stopSignal == nil {
		signaled := mr.updateWorkers(&currentWorkers, &workerIndex, startWorker, stopWorker)
		if signaled != nil {
			break
		}

		signaled = mr.updateConfig()
		if signaled != nil {
			break
		}
	}

	// Wait for workers to shutdown
	for currentWorkers > 0 {
		stopWorker <- true
		currentWorkers--
	}
	mr.log().Println("All workers stopped. Can exit now")
	mr.runFinished <- true
}

func (mr *RunCommand) interruptRun() {
	// Pump interrupt signal
	for {
		mr.runSignal <- mr.stopSignal
	}
}

func (mr *RunCommand) abortAllBuilds() {
	// Pump signal to abort all current builds
	for {
		mr.abortBuilds <- mr.stopSignal
	}
}

func (mr *RunCommand) handleGracefulShutdown() error {
	// We wait till we have a SIGQUIT
	for mr.stopSignal == syscall.SIGQUIT {
		mr.log().Warningln("Requested quit, waiting for builds to finish")

		// Wait for other signals to finish builds
		select {
		case mr.stopSignal = <-mr.stopSignals:
		// We received a new signal

		case <-mr.runFinished:
			// Everything finished we can exit now
			return nil
		}
	}

	return fmt.Errorf("received: %v", mr.stopSignal)
}

func (mr *RunCommand) handleShutdown() error {
	mr.log().Warningln("Requested service stop:", mr.stopSignal)

	go mr.abortAllBuilds()

	// Wait for graceful shutdown or abort after timeout
	for {
		select {
		case mr.stopSignal = <-mr.stopSignals:
			return fmt.Errorf("forced exit: %v", mr.stopSignal)

		case <-time.After(common.ShutdownTimeout * time.Second):
			return errors.New("shutdown timedout")

		case <-mr.runFinished:
			// Everything finished we can exit now
			return nil
		}
	}
}

func (mr *RunCommand) Stop(s service.Service) (err error) {
	go mr.interruptRun()
	err = mr.handleGracefulShutdown()
	if err == nil {
		return
	}
	err = mr.handleShutdown()
	return
}

func (mr *RunCommand) Execute(context *cli.Context) {
	svcConfig := &service.Config{
		Name:        mr.ServiceName,
		DisplayName: mr.ServiceName,
		Description: defaultDescription,
		Arguments:   []string{"run"},
		Option: service.KeyValue{
			"RunWait": mr.runWait,
		},
	}

	service, err := service_helpers.New(mr, svcConfig)
	if err != nil {
		log.Fatalln(err)
	}

	if mr.Syslog {
		log.SetFormatter(new(log.TextFormatter))
		logger, err := service.SystemLogger(nil)
		if err == nil {
			log.AddHook(&ServiceLogHook{logger, log.InfoLevel})
		} else {
			log.Errorln(err)
		}
	}

	log.AddHook(&mr.sentryLogHook)

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
