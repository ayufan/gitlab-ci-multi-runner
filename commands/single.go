package commands

import (
	"github.com/codegangsta/cli"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"os/signal"
	"syscall"
)

type RunSingleCommand struct {
	common.RunnerConfig
}

func (r *RunSingleCommand) Execute(c *cli.Context) {
	if len(r.URL) == 0 {
		log.Fatalln("Missing URL")
	}
	if len(r.Token) == 0 {
		log.Fatalln("Missing Token")
	}
	if len(r.Executor) == 0 {
		log.Fatalln("Missing Executor")
	}

	config := common.NewConfig()
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	log.Println("Starting runner for", r.URL, "with token", r.ShortDescription(), "...")

	finished := false
	abortSignal := make(chan os.Signal)
	doneSignal := make(chan int, 1)

	go func() {
		interrupt := <-signals
		finished = true

		// request stop, but wait for force exit
		for interrupt == syscall.SIGQUIT {
			log.Warningln("Requested quit, waiting for builds to finish")
			interrupt = <-signals
		}

		log.Warningln("Requested exit:", interrupt)

		go func() {
			for {
				abortSignal <- interrupt
			}
		}()

		select {
		case newSignal := <-signals:
			log.Fatalln("forced exit:", newSignal)
		case <-time.After(common.ShutdownTimeout * time.Second):
			log.Fatalln("shutdown timedout")
		case <-doneSignal:
		}
	}()

	for !finished {
		buildData, healthy := common.GetBuild(r.RunnerConfig)
		if !healthy {
			log.Println("Runner is not healthy!")
			select {
			case <-time.After(common.NotHealthyCheckInterval * time.Second):
			case <-abortSignal:
			}
			continue
		}

		if buildData == nil {
			select {
			case <-time.After(common.CheckInterval * time.Second):
			case <-abortSignal:
			}
			continue
		}

		newBuild := common.Build{
			GetBuildResponse: *buildData,
			Runner:           &r.RunnerConfig,
			BuildAbort:       abortSignal,
		}
		newBuild.AssignID()
		newBuild.Run(config)
	}

	doneSignal <- 0
}

func init() {
	common.RegisterCommand2("run-single", "start single runner", &RunSingleCommand{})
}
