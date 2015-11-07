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

func (e *AbstractExecutor) Debugln(args ...interface{}) {
	args = append([]interface{}{e.Config.ShortDescription(), e.Build.ID}, args...)
	log.Debugln(args...)
}

func (e *AbstractExecutor) Println(args ...interface{}) {
	if e.Build != nil {
		e.Build.WriteString(fmt.Sprintln(args...))
	}

	if len(args) == 0 {
		return
	}

	args = append([]interface{}{e.Config.ShortDescription(), e.Build.ID}, args...)
	log.Println(args...)
}

func (e *AbstractExecutor) Infoln(args ...interface{}) {
	if e.Build != nil {
		e.Build.WriteString(helpers.ANSI_BOLD_GREEN + fmt.Sprintln(args...) + helpers.ANSI_RESET)
	}

	if len(args) == 0 {
		return
	}

	args = append([]interface{}{e.Config.ShortDescription(), e.Build.ID}, args...)
	log.Println(args...)
}

func (e *AbstractExecutor) Warningln(args ...interface{}) {
	// write to log file
	if e.Build != nil {
		e.Build.WriteString(helpers.ANSI_BOLD_YELLOW + "WARNING: " + fmt.Sprintln(args...) + helpers.ANSI_RESET)
	}

	args = append([]interface{}{e.Config.ShortDescription(), e.Build.ID}, args...)
	log.Warningln(args...)
}

func (e *AbstractExecutor) Errorln(args ...interface{}) {
	// write to log file
	if e.Build != nil {
		e.Build.WriteString(helpers.ANSI_BOLD_RED + "ERROR: " + fmt.Sprintln(args...) + helpers.ANSI_RESET)
	}

	args = append([]interface{}{e.Config.ShortDescription(), e.Build.ID}, args...)
	log.Errorln(args...)
}

func (e *AbstractExecutor) readTrace(pipe *io.PipeReader) {
	defer e.Debugln("ReadTrace finished")

	traceStopped := false
	traceOutputLimit := helpers.NonZeroOrDefault(e.Config.OutputLimit, common.DefaultOutputLimit)
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

func (e *AbstractExecutor) updateTrace(config common.RunnerConfig, canceled chan bool, finished chan bool) {
	defer e.Debugln("PushTrace finished")

	buildLog := e.BuildLog
	if buildLog == nil {
		<-finished
		return
	}

	lastSentTrace := -1
	lastSentTime := time.Now()

	for {
		select {
		case <-time.After(common.UpdateInterval):
			// check if build log changed
			buildTraceLen := e.Build.BuildLogLen()
			if buildTraceLen == lastSentTrace && time.Since(lastSentTime) < common.ForceTraceSentInterval {
				e.Debugln("updateBuildLog", "Nothing to send.")
				continue
			}

			buildTrace := e.Build.BuildLog()
			switch e.Build.Network.UpdateBuild(config, e.Build.ID, common.Running, buildTrace) {
			case common.UpdateSucceeded:
				lastSentTrace = buildTraceLen
				lastSentTime = time.Now()

			case common.UpdateAbort:
				e.Debugln("updateBuildLog", "Sending abort request...")
				canceled <- true
				e.Debugln("updateBuildLog", "Waiting for finished flag...")
				<-finished
				e.Debugln("updateBuildLog", "Thread finished.")
				return
			case common.UpdateFailed:
			}

		case <-finished:
			e.Debugln("updateBuildLog", "Received finish.")
			return
		}
	}
}
