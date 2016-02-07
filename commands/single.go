package commands

import (
	"github.com/codegangsta/cli"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/network"
	"os/signal"
	"syscall"
)

type RunSingleCommand struct {
	common.RunnerConfig
	network common.Network
}

func waitForInterrupts(finished *bool, abortSignal chan os.Signal, doneSignal chan int) {
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	interrupt := <-signals
	if finished != nil {
		*finished = true
	}

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
}

func (r *RunSingleCommand) processBuild(abortSignal chan os.Signal) {
	buildData, _ := r.network.GetBuild(r.RunnerConfig)
	if buildData == nil {
		select {
		case <-time.After(common.CheckInterval * time.Second):
		case <-abortSignal:
		}
		return
	}

	config := common.NewConfig()

	newBuild := common.Build{
		GetBuildResponse: *buildData,
		Runner:           &r.RunnerConfig,
		BuildAbort:       abortSignal,
		Network:          r.network,
	}
	newBuild.AssignID()
	newBuild.Run(config)
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

	log.Println("Starting runner for", r.URL, "with token", r.ShortDescription(), "...")

	finished := false
	abortSignal := make(chan os.Signal)
	doneSignal := make(chan int, 1)

	go waitForInterrupts(&finished, abortSignal, doneSignal)

	for !finished {
		r.processBuild(abortSignal)
	}

	doneSignal <- 0
}

func init() {
	common.RegisterCommand2("run-single", "start single runner", &RunSingleCommand{
		network: &network.GitLabClient{},
	})
}
