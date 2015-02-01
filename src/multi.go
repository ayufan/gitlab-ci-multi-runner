package src

import (
	"fmt"
	"sync"
	"time"

	"github.com/codegangsta/cli"

	log "github.com/Sirupsen/logrus"
)

type RunnerHealth struct {
	failures  int
	lastCheck time.Time
}

type MultiRunner struct {
	config      *Config
	allBuilds   []*Build
	builds      []*Build
	buildsLock  sync.RWMutex
	healthy     map[string]*RunnerHealth
	healthyLock sync.Mutex
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

func (mr *MultiRunner) getHealth(runner *RunnerConfig) *RunnerHealth {
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

func (mr *MultiRunner) isHealthy(runner *RunnerConfig) bool {
	health := mr.getHealth(runner)
	if health.failures < HEALTHY_CHECKS {
		return true
	}

	if time.Since(health.lastCheck) > HEALTH_CHECK_INTERVAL*time.Second {
		mr.errorln("Runner", runner.ShortDescription(), "is not healthy, but will be checked!")
		health.failures = 0
		health.lastCheck = time.Now()
		return true
	}

	return false
}

func (mr *MultiRunner) makeHealthy(runner *RunnerConfig) {
	health := mr.getHealth(runner)
	health.failures = 0
	health.lastCheck = time.Now()
}

func (mr *MultiRunner) makeUnhealthy(runner *RunnerConfig) {
	health := mr.getHealth(runner)
	health.failures++

	if health.failures >= HEALTHY_CHECKS {
		mr.errorln("Runner", runner.ShortDescription(), "is not healthy and will be disabled!")
	}
}

func (mr *MultiRunner) addBuild(newBuild *Build) {
	mr.buildsLock.Lock()
	defer mr.buildsLock.Unlock()

	mr.builds = append(mr.builds, newBuild)
	mr.allBuilds = append(mr.allBuilds, newBuild)
	mr.debugln("Added a new build", newBuild)
}

func (mr *MultiRunner) removeBuild(deleteBuild *Build) bool {
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

func (mr *MultiRunner) buildsForRunner(runner *RunnerConfig) int {
	count := 0
	for _, build := range mr.builds {
		if build.Runner == runner {
			count += 1
		}
	}
	return count
}

func (mr *MultiRunner) requestBuild(runner *RunnerConfig) *Build {
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

	build_data, healthy := GetBuild(*runner)
	if healthy {
		mr.makeHealthy(runner)
	} else {
		mr.makeUnhealthy(runner)
	}

	if build_data == nil {
		return nil
	}

	mr.debugln("Received new build for", runner.ShortDescription(), "build", build_data.Id)
	new_build := &Build{
		GetBuildResponse: *build_data,
		Runner:           runner,
	}

	build_prefix := fmt.Sprintf("runner-%s", runner.ShortDescription())
	new_build.GenerateUniqueName(build_prefix, mr.builds)
	return new_build
}

func (mr *MultiRunner) feedRunners(runners chan *RunnerConfig) {
	for {
		mr.debugln("Feeding runners to channel")
		config := mr.config
		for _, runner := range config.Runners {
			runners <- runner
		}
		time.Sleep(CHECK_INTERVAL * time.Second)
	}
}

func (mr *MultiRunner) processRunners(id int, stop_worker chan bool, runners chan *RunnerConfig) {
	mr.debugln("Starting worker", id)
	for {
		select {
		case runner := <-runners:
			mr.debugln("Checking runner", runner, "on", id)
			new_job := mr.requestBuild(runner)
			if new_job == nil {
				break
			}

			mr.addBuild(new_job)
			new_job.Run()
			mr.removeBuild(new_job)

		case <-stop_worker:
			mr.debugln("Stopping worker", id)
			return
		}
	}
}

func (mr *MultiRunner) startWorkers(start_worker chan int, stop_worker chan bool, runners chan *RunnerConfig) {
	for {
		id := <-start_worker
		go mr.processRunners(id, stop_worker, runners)
	}
}

func runMulti(c *cli.Context) {
	mr := MultiRunner{}
	mr.config = &Config{}
	mr.allBuilds = []*Build{}
	mr.builds = []*Build{}
	err := mr.config.LoadConfig(c.String("config"))
	if err != nil {
		panic(err)
	}

	mr.config.SetChdir()
	mr.println("Starting multi-runner from", c.String("config"), "...")

	reload_config := make(chan Config)
	go ReloadConfig(c.String("config"), mr.config.ModTime, reload_config)

	runners := make(chan *RunnerConfig)
	go mr.feedRunners(runners)

	start_worker := make(chan int)
	stop_worker := make(chan bool)
	go mr.startWorkers(start_worker, stop_worker, runners)

	current_workers := 0
	worker_index := 0

	for {
		build_limit := mr.config.Concurrent

		for current_workers > build_limit {
			stop_worker <- true
			current_workers--
		}

		for current_workers < build_limit {
			start_worker <- worker_index
			current_workers++
			worker_index++
		}

		new_config := <-reload_config
		new_config.SetChdir()

		mr.debugln("Config reloaded.")
		mr.healthy = nil
		mr.config = &new_config
	}
}
