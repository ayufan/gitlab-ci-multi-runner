package commands

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/codegangsta/cli"

	log "github.com/Sirupsen/logrus"

	"github.com/ayufan/gitlab-ci-multi-runner/common"
)

type RunnerHealth struct {
	failures  int
	lastCheck time.Time
}

type MultiRunner struct {
	config      *common.Config
	configFile  string
	allBuilds   []*common.Build
	builds      []*common.Build
	buildsLock  sync.RWMutex
	healthy     map[string]*RunnerHealth
	healthyLock sync.Mutex
	finished    bool
	abortBuilds chan os.Signal
}

func (mr *MultiRunner) errorln(args ...interface{}) {
	args = append([]interface{}{len(mr.builds)}, args...)
	log.Errorln(args...)
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
	if health.failures < common.HEALTHY_CHECKS {
		return true
	}

	if time.Since(health.lastCheck) > common.HEALTH_CHECK_INTERVAL*time.Second {
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

	if health.failures >= common.HEALTHY_CHECKS {
		mr.errorln("Runner", runner.ShortDescription(), "is not healthy and will be disabled!")
	}
}

func (mr *MultiRunner) addBuild(newBuild *common.Build) {
	mr.buildsLock.Lock()
	defer mr.buildsLock.Unlock()

	mr.builds = append(mr.builds, newBuild)
	mr.allBuilds = append(mr.allBuilds, newBuild)
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
	if runner.Limit > 0 && count >= runner.Limit {
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
	}

	newBuild.Prepare(mr.builds)
	newBuild.BuildAbort = mr.abortBuilds
	return newBuild
}

func (mr *MultiRunner) feedRunners(runners chan *common.RunnerConfig) {
	for !mr.finished {
		mr.debugln("Feeding runners to channel")
		config := mr.config
		for _, runner := range config.Runners {
			runners <- runner
		}
		time.Sleep(common.CHECK_INTERVAL * time.Second)
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
	newConfig := common.Config{}
	err := newConfig.LoadConfig(mr.configFile)
	if err != nil {
		return err
	}

	newConfig.SetChdir()
	mr.healthy = nil
	mr.config = &newConfig
	return nil
}

func RunMulti(c *cli.Context) {
	mr := MultiRunner{
		configFile:  c.String("config"),
		allBuilds:   []*common.Build{},
		builds:      []*common.Build{},
		abortBuilds: make(chan os.Signal),
	}

	mr.println("Starting multi-runner from", mr.configFile, "...")

	err := mr.loadConfig()
	if err != nil {
		panic(err)
	}

	// Start webserver
	if listenAddr := c.String("listen-addr"); len(listenAddr) > 0 {
		mrs := MultiRunnerServer{
			MultiRunner:     &mr,
			listenAddresses: []string{listenAddr},
		}

		go mrs.Run()
	}

	runners := make(chan *common.RunnerConfig)
	go mr.feedRunners(runners)

	startWorker := make(chan int)
	stopWorker := make(chan bool)
	go mr.startWorkers(startWorker, stopWorker, runners)

	interruptSignal := make(chan os.Signal, 2)
	signal.Notify(interruptSignal, os.Interrupt, syscall.SIGTERM)

	reloadSignal := make(chan os.Signal, 1)
	signal.Notify(reloadSignal, syscall.SIGHUP)

	currentWorkers := 0
	workerIndex := 0

	var signaled os.Signal

finish_worker:
	for {
		buildLimit := mr.config.Concurrent

		for currentWorkers > buildLimit {
			select {
			case stopWorker <- true:
			case signaled = <-interruptSignal:
				break finish_worker
			}
			currentWorkers--
		}

		for currentWorkers < buildLimit {
			select {
			case startWorker <- workerIndex:
			case signaled = <-interruptSignal:
				break finish_worker
			}
			currentWorkers++
			workerIndex++
		}

		select {
		case <-time.After(common.RELOAD_CONFIG_INTERVAL * time.Second):
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

		case <-reloadSignal:
			err := mr.loadConfig()
			if err != nil {
				mr.errorln("Failed to load config", err)
				break
			}

			mr.println("Config reloaded.")

		case signaled = <-interruptSignal:
			break finish_worker
		}
	}

	mr.errorln("Received signal:", signaled)
	mr.finished = true

	close := make(chan int)

	// Pump signal to abort all builds
	go func() {
		for {
			mr.abortBuilds <- signaled
		}
	}()

	// Watch for second signal which will force to close process
	go func() {
		newSignal := <-interruptSignal
		mr.errorln("Forced exit:", newSignal)
		close <- 1
	}()

	// Wait for workers to shutdown
	go func() {
		for currentWorkers > 0 {
			stopWorker <- true
			currentWorkers--
		}
		mr.println("All workers stopped. Can exit now")
		close <- 0
	}()

	// Timeout shutdown
	go func() {
		time.Sleep(common.SHUTDOWN_TIMEOUT * time.Second)
		mr.errorln("Shutdown timedout.")
		close <- 1
	}()

	os.Exit(<-close)
}

var (
	CmdRunMulti = cli.Command{
		Name:      "run",
		ShortName: "r",
		Usage:     "run multi runner",
		Action:    RunMulti,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "docker-host",
				Value:  "",
				Usage:  "Docker endpoint URL",
				EnvVar: "DOCKER_HOST",
			},
			cli.StringFlag{
				Name:   "config",
				Value:  "config.toml",
				Usage:  "Config file",
				EnvVar: "CONFIG_FILE",
			},
			cli.StringFlag{
				Name:   "listen-addr",
				Value:  "",
				Usage:  "API listen address, eg. :8080",
				EnvVar: "API_LISTEN",
			},
		},
	}
)
