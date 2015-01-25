package src

import (
	"fmt"
	"time"

	"github.com/codegangsta/cli"

	log "github.com/Sirupsen/logrus"
)

func requestNewJob(config *Config, jobs []*Job) (*GetBuildResponse, *RunnerConfig) {
	for _, runner := range config.Runners {
		if runner == nil {
			continue
		}

		count := 0
		for _, job := range jobs {
			if job.Runner == runner {
				count += 1
			}
		}

		if runner.Limit > 0 && count >= runner.Limit {
			continue
		}

		new_build := GetBuild(*runner)
		if new_build != nil {
			return new_build, runner
		}
	}

	return nil, nil
}

func startNewJob(config *Config, jobs []*Job, finish chan *Job) *Job {
	if config.Concurrent <= len(jobs) {
		return nil
	}

	log.Debugln(len(jobs), "Requesting a new job...")

	new_build, runner_config := requestNewJob(config, jobs)
	if new_build == nil {
		return nil
	}
	if runner_config == nil {
		// this shouldn't happen
		return nil
	}

	log.Debugln(len(jobs), "Received new job for", runner_config.ShortDescription(), "build", new_build.Id)
	new_job := &Job{
		Build: &Build{
			GetBuildResponse: *new_build,
			Name:             "",
		},
		Runner: runner_config,
		Finish: finish,
	}

	build_prefix := fmt.Sprintf("runner-%s-", runner_config.ShortDescription())

	other_builds := []*Build{}
	for _, other_job := range jobs {
		other_builds = append(other_builds, other_job.Build)
	}
	new_job.Build.GenerateUniqueName(build_prefix, other_builds)

	go new_job.Run()
	return new_job
}

func runMulti(c *cli.Context) {
	config, config_time, err := LoadConfig(c.String("config"))
	if err != nil {
		panic(err)
	}

	log.Println("Starting multi-runner from", c.String("config"), "...")

	jobs := []*Job{}
	job_finish := make(chan *Job)

	reload_config := make(chan Config)
	go ReloadConfig(c.String("config"), config_time, reload_config)

	for {
		new_job := startNewJob(&config, jobs, job_finish)
		if new_job != nil {
			jobs = append(jobs, new_job)
			log.Debugln(len(jobs), "Added a new job", new_job)
		}

		select {
		case finished_job := <-job_finish:
			log.Debugln(len(jobs), "Job finished", finished_job)
			for idx, job := range jobs {
				if job == finished_job {
					jobs[idx] = jobs[len(jobs)-1]
					jobs = jobs[:len(jobs)-1]
					log.Debugln(len(jobs), "Removed finished job", finished_job)
					break
				}
			}

		case new_config := <-reload_config:
			log.Debugln(len(jobs), "Config reloaded.")
			config = new_config

		case <-time.After(CHECK_INTERVAL * time.Second):
			log.Debugln(len(jobs), "Check interval fired")
		}
	}
}
