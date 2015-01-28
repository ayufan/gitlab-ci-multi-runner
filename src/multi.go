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
	jobs        []*Job
	jobsLock    sync.RWMutex
	healthy     map[string]*RunnerHealth
	healthyLock sync.Mutex
}

func (mr *MultiRunner) errorln(args ...interface{}) {
	args = append([]interface{}{len(mr.jobs)}, args...)
	log.Errorln(args...)
}

func (mr *MultiRunner) debugln(args ...interface{}) {
	args = append([]interface{}{len(mr.jobs)}, args...)
	log.Debugln(args...)
}

func (mr *MultiRunner) println(args ...interface{}) {
	args = append([]interface{}{len(mr.jobs)}, args...)
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

func (mr *MultiRunner) addJob(newJob *Job) {
	mr.jobsLock.Lock()
	defer mr.jobsLock.Unlock()

	mr.jobs = append(mr.jobs, newJob)
	mr.debugln("Added a new job", newJob)
}

func (mr *MultiRunner) removeJob(deleteJob *Job) bool {
	mr.jobsLock.Lock()
	defer mr.jobsLock.Unlock()

	for idx, job := range mr.jobs {
		if job == deleteJob {
			mr.jobs[idx] = mr.jobs[len(mr.jobs)-1]
			mr.jobs = mr.jobs[:len(mr.jobs)-1]
			mr.debugln("Removed job", deleteJob)
			return true
		}
	}
	return false
}

func (mr *MultiRunner) jobsForRunner(runner *RunnerConfig) int {
	mr.jobsLock.RLock()
	defer mr.jobsLock.RUnlock()

	count := 0
	for _, job := range mr.jobs {
		if job.Runner == runner {
			count += 1
		}
	}
	return count
}

func (mr *MultiRunner) getAllBuilds() []*Build {
	mr.jobsLock.RLock()
	defer mr.jobsLock.RUnlock()

	other_builds := []*Build{}
	for _, other_job := range mr.jobs {
		other_builds = append(other_builds, other_job.Build)
	}

	return other_builds
}

func (mr *MultiRunner) requestJob(runner *RunnerConfig) *Job {
	if runner == nil {
		return nil
	}

	if !mr.isHealthy(runner) {
		return nil
	}

	count := mr.jobsForRunner(runner)
	if runner.Limit > 0 && count >= runner.Limit {
		return nil
	}

	new_build, healthy := GetBuild(*runner)
	if healthy {
		mr.makeHealthy(runner)
	} else {
		mr.makeUnhealthy(runner)
	}

	if new_build == nil {
		return nil
	}

	log.Debugln(len(mr.jobs), "Received new job for", runner.ShortDescription(), "build", new_build.Id)
	new_job := &Job{
		Build: &Build{
			GetBuildResponse: *new_build,
		},
		Runner: runner,
	}

	build_prefix := fmt.Sprintf("runner-%s", runner.ShortDescription())
	new_job.Build.GenerateUniqueName(build_prefix, mr.getAllBuilds())
	return new_job
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
			new_job := mr.requestJob(runner)
			if new_job == nil {
				break
			}

			mr.addJob(new_job)
			new_job.Run()
			mr.removeJob(new_job)

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
		jobs_limit := mr.config.Concurrent

		for current_workers > jobs_limit {
			stop_worker <- true
			current_workers--
		}

		for current_workers < jobs_limit {
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
