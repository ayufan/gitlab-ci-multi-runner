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

func (r *RunSingleCommand) processBuild(data common.ExecutorData, abortSignal chan os.Signal) (err error) {
	buildData, healthy := r.network.GetBuild(r.RunnerConfig)
	if !healthy {
		log.Println("Runner is not healthy!")
		select {
		case <-time.After(common.NotHealthyCheckInterval * time.Second):
		case <-abortSignal:
		}
		return
	}

	if buildData == nil {
		select {
		case <-time.After(common.CheckInterval):
		case <-abortSignal:
		}
		return
	}

	config := common.NewConfig()

	newBuild := common.Build{
		GetBuildResponse: *buildData,
		Runner:           &r.RunnerConfig,
		BuildAbort:       abortSignal,
		ExecutorData:     data,
	}

	buildCredentials := &common.BuildCredentials{
		ID:    buildData.ID,
		Token: buildData.Token,
	}
	trace := r.network.ProcessBuild(r.RunnerConfig, buildCredentials)
	defer trace.Fail(err)

	err = newBuild.Run(config, trace)
	return
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

	executorProvider := common.GetExecutor(r.Executor)
	if executorProvider == nil {
		log.Fatalln("Uknown executor:", r.Executor)
	}

	log.Println("Starting runner for", r.URL, "with token", r.ShortDescription(), "...")

	finished := false
	abortSignal := make(chan os.Signal)
	doneSignal := make(chan int, 1)

	go waitForInterrupts(&finished, abortSignal, doneSignal)

	for !finished {
		data, err := executorProvider.Acquire(&r.RunnerConfig)
		if err != nil {
			log.Warningln("Executor update:", err)
		}

		r.processBuild(data, abortSignal)
		executorProvider.Release(&r.RunnerConfig, data)
	}

	doneSignal <- 0
}

func init() {
	common.RegisterCommand2("run-single", "start single runner", &RunSingleCommand{
		network: &network.GitLabClient{},
	})
}
