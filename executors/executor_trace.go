package executors

import (
	"bufio"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"io"
	"time"
)

func (e *AbstractExecutor) log() *log.Entry {
	return e.Config.Log().WithField("build", e.Build.ID)
}

func (e *AbstractExecutor) Debugln(args ...interface{}) {
	e.log().Debugln(args...)
}

func (e *AbstractExecutor) Println(args ...interface{}) {
	if e.Build != nil {
		e.Build.WriteString(fmt.Sprintln(args...))
	}

	if len(args) == 0 {
		return
	}

	e.log().Println(args...)
}

func (e *AbstractExecutor) Infoln(args ...interface{}) {
	if e.Build != nil {
		e.Build.WriteString(helpers.ANSI_BOLD_GREEN + fmt.Sprintln(args...) + helpers.ANSI_RESET)
	}

	if len(args) == 0 {
		return
	}

	e.log().Println(args...)
}

func (e *AbstractExecutor) Warningln(args ...interface{}) {
	// write to log file
	if e.Build != nil {
		e.Build.WriteString(helpers.ANSI_BOLD_YELLOW + "WARNING: " + fmt.Sprintln(args...) + helpers.ANSI_RESET)
	}

	e.log().Warningln(args...)
}

func (e *AbstractExecutor) Errorln(args ...interface{}) {
	// write to log file
	if e.Build != nil {
		e.Build.WriteString(helpers.ANSI_BOLD_RED + "ERROR: " + fmt.Sprintln(args...) + helpers.ANSI_RESET)
	}

	e.log().Errorln(args...)
}

func (e *AbstractExecutor) readTrace(pipe *io.PipeReader) {
	defer e.Debugln("ReadTrace finished")

	traceStopped := false
	traceOutputLimit := e.Config.OutputLimit
	if traceOutputLimit == 0 {
		traceOutputLimit = common.DefaultOutputLimit
	}
	traceOutputLimit *= 1024

	reader := bufio.NewReader(pipe)
	for {
		r, s, err := reader.ReadRune()
		if s <= 0 {
			break
		} else if traceStopped {
			// ignore symbols if build log exceeded limit
			continue
		} else if err == nil {
			e.Build.WriteRune(r)
		} else {
			// ignore invalid characters
			continue
		}

		if e.Build.BuildLogLen() > traceOutputLimit {
			output := fmt.Sprintf("\n%sBuild log exceeded limit of %v bytes.%s\n",
				helpers.ANSI_BOLD_RED,
				traceOutputLimit,
				helpers.ANSI_RESET,
			)
			e.Build.WriteString(output)
			traceStopped = true
		}
	}

	pipe.Close()
}

func (e *AbstractExecutor) uploadTrace(config *common.RunnerConfig, lastSentTrace *int, lastSentTime *time.Time) bool {
	// check if build log changed
	buildTraceLen := e.Build.BuildLogLen()
	if buildTraceLen == *lastSentTrace && time.Since(*lastSentTime) < common.ForceTraceSentInterval {
		e.Debugln("updateBuildLog", "Nothing to send.")
		return true
	}

	buildTrace := e.Build.BuildLog()
	switch e.Build.Network.UpdateBuild(*config, e.Build.ID, common.Running, buildTrace) {
	case common.UpdateSucceeded:
		*lastSentTrace = buildTraceLen
		*lastSentTime = time.Now()
		return true

	case common.UpdateAbort:
		return false

	default:
		return true
	}
}

func (e *AbstractExecutor) updateTrace(config common.RunnerConfig, canceled chan bool, finished chan bool) {
	defer e.Debugln("PushTrace finished")

	buildLog := e.BuildLog
	if buildLog == nil || e.Build.Network == nil {
		<-finished
		return
	}

	lastSentTrace := -1
	lastSentTime := time.Now()

	for {
		select {
		case <-time.After(common.UpdateInterval):
			continueSending := e.uploadTrace(&config, &lastSentTrace, &lastSentTime)
			if !continueSending {
				e.Debugln("updateBuildLog", "Sending abort request...")
				canceled <- true
				e.Debugln("updateBuildLog", "Waiting for finished flag...")
				<-finished
				e.Debugln("updateBuildLog", "Thread finished.")
				return
			}

		case <-finished:
			e.Debugln("updateBuildLog", "Received finish.")
			return
		}
	}
}
