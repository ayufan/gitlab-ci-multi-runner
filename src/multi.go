package src

import (
	"fmt"
	"time"

	"github.com/codegangsta/cli"

	log "github.com/Sirupsen/logrus"
)

type RunnerHealth struct {
	failures  int
	lastCheck time.Time
}

type MultiRunner struct {
	config  Config
	jobs    []*Job
	healthy map[string]*RunnerHealth
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

func (mr *MultiRunner) requestNewJob() (*GetBuildResponse, *RunnerConfig) {
	for _, runner := range mr.config.Runners {
		if runner == nil {
			continue
		}

		if !mr.isHealthy(runner) {
			continue
		}

		count := 0
		for _, job := range mr.jobs {
			if job.Runner == runner {
				count += 1
			}
		}

		if runner.Limit > 0 && count >= runner.Limit {
			continue
		}

		new_build, healthy := GetBuild(*runner)
		if new_build != nil {
			return new_build, runner
		}

		if healthy {
			mr.makeHealthy(runner)
		} else {
			mr.makeUnhealthy(runner)
		}
	}

	return nil, nil
}

func (mr *MultiRunner) startNewJob(finish chan *Job) *Job {
	if mr.config.Concurrent <= len(mr.jobs) {
		return nil
	}

	log.Debugln(len(mr.jobs), "Requesting a new job...")

	new_build, runner_config := mr.requestNewJob()
	if new_build == nil {
		return nil
	}
	if runner_config == nil {
		// this shouldn't happen
		return nil
	}

	log.Debugln(len(mr.jobs), "Received new job for", runner_config.ShortDescription(), "build", new_build.Id)
	new_job := &Job{
		Build: &Build{
			GetBuildResponse: *new_build,
		},
		Runner: runner_config,
	}

	build_prefix := fmt.Sprintf("runner-%s", runner_config.ShortDescription())

	other_builds := []*Build{}
	for _, other_job := range mr.jobs {
		other_builds = append(other_builds, other_job.Build)
	}
	new_job.Build.GenerateUniqueName(build_prefix, other_builds)

	go func() {
		new_job.Run()
		finish <- new_job
	}()
	return new_job
}

func (mr *MultiRunner) addJob(newJob *Job) {
	mr.jobs = append(mr.jobs, newJob)
	mr.debugln("Added a new job", newJob)
}

func (mr *MultiRunner) removeJob(deleteJob *Job) bool {
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

func runMulti(c *cli.Context) {
	mr := MultiRunner{}
	err := mr.config.LoadConfig(c.String("config"))
	if err != nil {
		panic(err)
	}

	mr.config.SetChdir()
	mr.println("Starting multi-runner from", c.String("config"), "...")

	job_finish := make(chan *Job)
	reload_config := make(chan Config)
	go ReloadConfig(c.String("config"), mr.config.ModTime, reload_config)

	for {
		new_job := mr.startNewJob(job_finish)
		if new_job != nil {
			mr.addJob(new_job)
		}

		select {
		case finished_job := <-job_finish:
			mr.debugln("Job finished", finished_job)
			mr.removeJob(finished_job)

		case new_config := <-reload_config:
			mr.debugln("Config reloaded.")
			mr.healthy = nil
			mr.config = new_config
			mr.config.SetChdir()

		case <-time.After(CHECK_INTERVAL * time.Second):
			mr.debugln("Check interval fired")
		}
	}
}
