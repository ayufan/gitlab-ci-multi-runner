package executors

import (
	"fmt"
	"os"
	"time"

	"bufio"
	log "github.com/Sirupsen/logrus"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"io"
	"path/filepath"
)

type AbstractExecutor struct {
	DefaultBuildsDir string
	SharedBuildsDir  bool
	DefaultShell     string
	ShellType        common.ShellType
	ShowHostname     bool
	Config           *common.RunnerConfig
	Build            *common.Build
	BuildCanceled    chan bool
	FinishLogWatcher chan bool
	BuildFinish      chan error
	BuildLog         *io.PipeWriter
	ShellScript      *common.ShellScript
}

func (e *AbstractExecutor) ReadTrace(pipe *io.PipeReader) {
	defer e.Debugln("ReadTrace finished")

	reader := bufio.NewReader(pipe)
	for {
		r, s, err := reader.ReadRune()
		if s <= 0 {
			break
		} else if err == nil {
			e.Build.WriteRune(r)
		} else {
			// ignore invalid characters
			continue
		}

		if e.Build.BuildLogLen() > common.MaxTraceOutputSize {
			output := fmt.Sprintf("\nBuild log exceed limit of %v bytes.", common.MaxTraceOutputSize)
			e.Build.WriteString(output)
			break
		}
	}

	pipe.Close()
}

func (e *AbstractExecutor) PushTrace(config common.RunnerConfig, canceled chan bool, finished chan bool) {
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
		case <-time.After(common.UpdateInterval * time.Second):
			// check if build log changed
			buildTraceLen := e.Build.BuildLogLen()
			if buildTraceLen == lastSentTrace && time.Since(lastSentTime) > common.ForceTraceSentInterval {
				e.Debugln("updateBuildLog", "Nothing to send.")
				continue
			}

			buildTrace := e.Build.BuildLog()
			switch common.UpdateBuild(config, e.Build.ID, common.Running, buildTrace) {
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

func (e *AbstractExecutor) Debugln(args ...interface{}) {
	args = append([]interface{}{e.Config.ShortDescription(), e.Build.ID}, args...)
	log.Debugln(args...)
}

func (e *AbstractExecutor) Println(args ...interface{}) {
	if e.Build != nil {
		e.Build.WriteString(fmt.Sprintln(args...))
	}

	args = append([]interface{}{e.Config.ShortDescription(), e.Build.ID}, args...)
	log.Println(args...)
}

func (e *AbstractExecutor) Errorln(args ...interface{}) {
	// write to log file
	if e.Build != nil {
		e.Build.WriteString(fmt.Sprintln(args...))
	}

	args = append([]interface{}{e.Config.ShortDescription(), e.Build.ID}, args...)
	log.Errorln(args...)
}

func (e *AbstractExecutor) generateShellScript() error {
	shell := helpers.StringOrDefault(e.Config.Shell, e.DefaultShell)
	shellScript, err := common.GenerateShellScript(shell, e.Build, e.ShellType)
	if err != nil {
		return err
	}
	e.ShellScript = shellScript
	e.Debugln("Shell script:", shellScript)
	return nil
}

func (e *AbstractExecutor) startBuild() error {
	// Craete pipe where data are read
	reader, writer := io.Pipe()
	go e.ReadTrace(reader)
	e.BuildLog = writer

	// Save hostname
	if e.ShowHostname {
		e.Build.Hostname, _ = os.Hostname()
	}

	// Deduce build directory
	buildsDir := helpers.StringOrDefault(e.Config.BuildsDir, e.DefaultBuildsDir)

	if e.SharedBuildsDir {
		buildsDir = filepath.Join(buildsDir, e.Build.ProjectUniqueName())
	}
	if slug, err := e.Build.ProjectSlug(); err == nil {
		buildsDir = filepath.Join(buildsDir, slug)
	}

	// Start actual build
	e.Build.StartBuild(buildsDir)
	return nil
}

func (e *AbstractExecutor) Prepare(config *common.RunnerConfig, build *common.Build) error {
	e.Config = config
	e.Build = build
	e.BuildCanceled = make(chan bool, 1)
	e.BuildFinish = make(chan error, 1)
	e.FinishLogWatcher = make(chan bool)

	err := e.startBuild()
	if err != nil {
		return err
	}

	e.Println(fmt.Sprintf("%s %s (%s)", common.NAME, common.VERSION, common.REVISION))

	err = e.generateShellScript()
	if err != nil {
		return err
	}

	go e.PushTrace(*e.Config, e.BuildCanceled, e.FinishLogWatcher)
	return nil
}

func (e *AbstractExecutor) Wait() error {
	e.Build.BuildState = common.Running

	buildTimeout := e.Build.Timeout
	if buildTimeout <= 0 {
		buildTimeout = common.DefaultTimeout
	}

	// Wait for signals: cancel, timeout, abort or finish
	log.Debugln(e.Config.ShortDescription(), e.Build.ID, "Waiting for signals...")
	select {
	case <-e.BuildCanceled:
		log.Println(e.Config.ShortDescription(), e.Build.ID, "Build got canceled.")
		e.Build.FinishBuild(common.Failed, "Build got canceled")

	case <-time.After(time.Duration(buildTimeout) * time.Second):
		log.Println(e.Config.ShortDescription(), e.Build.ID, "Build timedout.")
		e.Build.FinishBuild(common.Failed, "CI Timeout. Execution took longer then %d seconds", buildTimeout)

	case signal := <-e.Build.BuildAbort:
		log.Println(e.Config.ShortDescription(), e.Build.ID, "Build got aborted", signal)
		e.Build.FinishBuild(common.Failed, "Build got aborted: %v", signal)

	case err := <-e.BuildFinish:
		if err != nil {
			return err
		}

		log.Println(e.Config.ShortDescription(), e.Build.ID, "Build succeeded.")
		e.Build.FinishBuild(common.Success, "Build succeeded.")
	}
	return nil
}

func (e *AbstractExecutor) Finish(err error) {
	if err != nil {
		e.Build.FinishBuild(common.Failed, "Build failed with %v", err)
	}

	e.Debugln("Build took", e.Build.BuildDuration)

	if e.BuildLog != nil {
		// wait for update log routine to finish
		e.Debugln("Waiting for build log updater to finish")
		e.FinishLogWatcher <- true
		e.Debugln("Build log updater finished.")
	}

	// Send final build state to server
	e.Build.SendBuildLog()
	e.Println("Build finished.")
}

func (e *AbstractExecutor) Cleanup() {
	if e.BuildLog != nil {
		e.BuildLog.Close()
	}
}
