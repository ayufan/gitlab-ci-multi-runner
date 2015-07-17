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
	"strings"
)

type ExecutorOptions struct {
	DefaultBuildsDir string
	SharedBuildsDir  bool
	Shell            common.ShellScriptInfo
	ShowHostname     bool
	SupportedOptions []string
}

type AbstractExecutor struct {
	ExecutorOptions
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
		e.Build.WriteString(helpers.ANSI_BOLD_YELLOW + "WARNING:" + fmt.Sprintln(args...) + helpers.ANSI_RESET)
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

func (e *AbstractExecutor) generateShellScript() error {
	script := &e.Shell
	script.Build = e.Build
	script.Shell = helpers.StringOrDefault(e.Config.Shell, script.Shell)

	// Add config variables
	for _, environment := range e.Config.Environment {
		keyValue := strings.SplitN(environment, "=", 2)
		if len(keyValue) != 2 {
			continue
		}
		variable := common.BuildVariable{
			Key: keyValue[0],
			Value: keyValue[1],
		}
		script.Environment = append(script.Environment, variable)
	}

	// Add secure variables
	script.Environment = append(script.Environment, e.Build.Variables...)

	// Generate shell script
	shellScript, err := common.GenerateShellScript(*script)
	if err != nil {
		return err
	}
	e.ShellScript = shellScript
	e.Debugln("Shell script:", shellScript)
	return nil
}

func (e *AbstractExecutor) startBuild() error {
	// Create pipe where data are read
	reader, writer := io.Pipe()
	go e.ReadTrace(reader)
	e.BuildLog = writer

	// Save hostname
	if e.ShowHostname {
		e.Build.Hostname, _ = os.Hostname()
	}

	// Start actual build
	rootDir := helpers.StringOrDefault(e.Config.BuildsDir, e.DefaultBuildsDir)
	e.Build.StartBuild(rootDir, e.SharedBuildsDir)
	return nil
}

func (e *AbstractExecutor) verifyOptions() error {
	for key, _ := range e.Build.Options {
		found := false
		for _, option := range e.SupportedOptions {
			if option == key {
				found = true
				break
			}
		}

		if !found {
			e.Warningln("Defined '%s' is not supported for that executor", key)
		}
	}
	return nil
}

func (e *AbstractExecutor) Prepare(globalConfig *common.Config, config *common.RunnerConfig, build *common.Build) error {
	e.Config = config
	e.Build = build
	e.BuildCanceled = make(chan bool, 1)
	e.BuildFinish = make(chan error, 1)
	e.FinishLogWatcher = make(chan bool)

	err := e.startBuild()
	if err != nil {
		return err
	}

	e.Infoln(fmt.Sprintf("%s %s (%s)", common.NAME, common.VERSION, common.REVISION))

	err = e.verifyOptions()
	if err != nil {
		return err
	}

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
		e.Println()
		e.Warningln("Build got canceled.")
		e.Build.FinishBuild(common.Failed)

	case <-time.After(time.Duration(buildTimeout) * time.Second):
		e.Println()
		e.Errorln("CI Timeout. Execution took longer then", buildTimeout, "seconds.")
		e.Build.FinishBuild(common.Failed)

	case signal := <-e.Build.BuildAbort:
		e.Println()
		e.Errorln("Build got aborted:", signal)
		e.Build.FinishBuild(common.Failed)

	case err := <-e.BuildFinish:
		if err != nil {
			return err
		}

		e.Println()
		e.Infoln("Build succeeded.")
		e.Build.FinishBuild(common.Success)
	}
	return nil
}

func (e *AbstractExecutor) Finish(err error) {
	if err != nil {
		e.Println()
		e.Errorln("Build failed with:", err)
		e.Build.FinishBuild(common.Failed)
	}

	e.Debugln("Build took", e.Build.BuildDuration)

	if e.BuildLog != nil {
		// wait for update log routine to finish
		e.Debugln("Waiting for build log updater to finish")
		e.FinishLogWatcher <- true
		e.Debugln("Build log updater finished.")
	}

	e.Debugln("Build log: ", e.Build.BuildLog())

	// Send final build state to server
	e.Build.SendBuildLog()
	e.Println("Build finished.")
}

func (e *AbstractExecutor) Cleanup() {
	if e.BuildLog != nil {
		e.BuildLog.Close()
	}
}
