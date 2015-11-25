package shells

import (
	"bufio"
	"bytes"
	"fmt"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"io"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type BashShell struct {
	AbstractShell
}

func (b *BashShell) GetName() string {
	return "bash"
}

func (b *BashShell) GetFeatures(features *common.FeaturesInfo) {
	features.Artifacts = true
	features.Cache = true
}

func (b *BashShell) executeCommand(w io.Writer, cmd string, arguments ...string) {
	list := []string{
		helpers.ShellEscape(cmd),
	}

	for _, argument := range arguments {
		list = append(list, strconv.Quote(argument))
	}

	io.WriteString(w, strings.Join(list, " ")+"\n")
}

func (b *BashShell) executeCommandFormat(w io.Writer, format string, arguments ...interface{}) {
	io.WriteString(w, fmt.Sprintf(format+"\n", arguments...))
}

func (b *BashShell) echoColored(w io.Writer, text string) {
	coloredText := helpers.ANSI_BOLD_GREEN + text + helpers.ANSI_RESET
	b.executeCommandFormat(w, "echo %s", helpers.ShellEscape(coloredText))
}

func (b *BashShell) echoWarning(w io.Writer, text string) {
	coloredText := helpers.ANSI_BOLD_YELLOW + text + helpers.ANSI_RESET
	b.executeCommandFormat(w, "echo %s", helpers.ShellEscape(coloredText))
}

func (b *BashShell) echoColoredFormat(w io.Writer, format string, a ...interface{}) {
	b.echoColored(w, fmt.Sprintf(format, a...))
}

func (b *BashShell) writeIfDirectory(w io.Writer, directory string) {
	b.executeCommandFormat(w, "if [[ -d %q ]]; then", directory)
}

func (b *BashShell) writeIfFile(w io.Writer, directory string) {
	b.executeCommandFormat(w, "if [[ -e %q ]]; then", directory)
}

func (b *BashShell) writeElse(w io.Writer) {
	b.executeCommandFormat(w, "else")
}

func (b *BashShell) writeEndIf(w io.Writer) {
	b.executeCommandFormat(w, "fi")
}

func (b *BashShell) writeCloneCmd(w io.Writer, build *common.Build, projectDir string) {
	b.echoColoredFormat(w, "Cloning repository...")
	b.executeCommand(w, "rm", "-rf", projectDir)
	b.executeCommand(w, "mkdir", "-p", projectDir)
	b.executeCommand(w, "git", "clone", build.RepoURL, projectDir)
	b.executeCommand(w, "cd", projectDir)
}

func (b *BashShell) writeFetchCmd(w io.Writer, build *common.Build, projectDir string, gitDir string) {
	b.writeIfDirectory(w, gitDir)
	b.echoColoredFormat(w, "Fetching changes...")
	b.executeCommand(w, "cd", projectDir)
	b.executeCommand(w, "git", "clean", "-ffdx")
	b.executeCommand(w, "git", "reset", "--hard")
	b.executeCommand(w, "git", "remote", "set-url", "origin", build.RepoURL)
	b.executeCommand(w, "git", "fetch", "origin")
	b.writeElse(w)
	b.writeCloneCmd(w, build, projectDir)
	b.writeEndIf(w)
}

func (b *BashShell) writeCheckoutCmd(w io.Writer, build *common.Build) {
	b.echoColoredFormat(w, "Checking out %s as %s...", build.Sha[0:8], build.RefName)
	b.executeCommand(w, "git", "checkout", build.Sha)
}

func (b *BashShell) writeCdBuildDir(w io.Writer, info common.ShellScriptInfo) {
	b.executeCommand(w, "cd", b.fullProjectDir(info))
}

func (b *BashShell) writeExport(w io.Writer, variable common.BuildVariable) {
	b.executeCommandFormat(w, "export %s=%s", helpers.ShellEscape(variable.Key), helpers.ShellEscape(variable.Value))
}

func (b *BashShell) fullProjectDir(info common.ShellScriptInfo) string {
	projectDir := info.Build.FullProjectDir()
	return helpers.ToSlash(projectDir)
}

func (b *BashShell) writeExports(w io.Writer, info common.ShellScriptInfo) {
	for _, variable := range info.Build.GetAllVariables() {
		b.writeExport(w, variable)
	}
}

func (b *BashShell) generatePreBuildScript(info common.ShellScriptInfo) string {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)

	b.writeExports(w, info)

	if len(info.Build.Hostname) != 0 {
		b.executeCommandFormat(w, "echo %q", "Running on $(hostname) via "+info.Build.Hostname+"...")
	} else {
		b.executeCommandFormat(w, "echo %q", "Running on $(hostname)...")
	}

	build := info.Build
	projectDir := b.fullProjectDir(info)
	gitDir := filepath.Join(projectDir, ".git")

	if build.AllowGitFetch {
		b.writeFetchCmd(w, build, projectDir, gitDir)
	} else {
		b.writeCloneCmd(w, build, projectDir)
	}

	b.writeCheckoutCmd(w, build)

	cacheFile := info.Build.CacheFile()
	cacheFile2 := info.Build.CacheFileForRef("master")
	if cacheFile == "" {
		cacheFile = cacheFile2
		cacheFile2 = ""
	}

	// Try to restore from main cache, if not found cache for master
	if cacheFile != "" {
		// If we have cache, restore it
		b.writeIfFile(w, cacheFile)
		b.echoColored(w, "Restoring cache...")
		b.extractFiles(w, info.RunnerCommand, "cache", cacheFile)
		if cacheFile2 != "" {
			b.writeElse(w)

			// If we have cache, restore it
			b.writeIfFile(w, cacheFile2)
			b.echoColored(w, "Restoring cache...")
			b.extractFiles(w, info.RunnerCommand, "cache", cacheFile2)
			b.writeEndIf(w)
		}
		b.writeEndIf(w)
	}

	w.Flush()

	return b.finalize(buffer.String())
}

func (b *BashShell) generateCommands(info common.ShellScriptInfo) string {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)

	b.writeExports(w, info)
	b.writeCdBuildDir(w, info)

	commands := info.Build.Commands
	commands = strings.TrimSpace(commands)
	for _, command := range strings.Split(commands, "\n") {
		command = strings.TrimSpace(command)
		if !helpers.BoolOrDefault(info.Build.Runner.DisableVerbose, false) {
			if command != "" {
				b.echoColored(w, "$ "+command)
			} else {
				b.executeCommand(w, "echo")
			}
		}
		io.WriteString(w, command+"\n")
	}

	w.Flush()

	return b.finalize(buffer.String())
}

func (b *BashShell) extractFiles(w io.Writer, runnerCommand, archiveType, archivePath string) {
	args := []string{
		"extract",
		"--silent",
		"--input",
		archivePath,
	}

	// Execute extract command
	b.echoColoredFormat(w, "Extracting %s...", archiveType)
	if runnerCommand == "" {
		runnerCommand = "gitlab-runner"
	}
	b.executeCommand(w, runnerCommand, args...)
}

func (b *BashShell) archiveFiles(w io.Writer, list interface{}, runnerCommand, archiveType, archivePath string) {
	hash, ok := list.(map[string]interface{})
	if !ok {
		return
	}

	args := []string{
		"archive",
		"--silent",
		"--output",
		archivePath,
	}

	// Collect paths
	if paths, ok := hash["paths"].([]interface{}); ok {
		for _, artifactPath := range paths {
			if file, ok := artifactPath.(string); ok {
				args = append(args, "--path", file)
			}
		}
	}

	// Archive also untracked files
	if untracked, ok := hash["untracked"].(bool); ok && untracked {
		args = append(args, "--untracked")
	}

	// Skip creating archive
	if len(args) <= 3 {
		return
	}

	// Execute archive command
	b.echoColoredFormat(w, "Archiving %s...", archiveType)
	if runnerCommand == "" {
		runnerCommand = "gitlab-runner"
	}
	b.executeCommand(w, runnerCommand, args...)
}

func (b *BashShell) generatePostBuildScript(info common.ShellScriptInfo) string {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)

	b.writeExports(w, info)
	b.writeCdBuildDir(w, info)

	// Find cached files and archive them
	if cacheFile := info.Build.CacheFile(); cacheFile != "" {
		b.archiveFiles(w, info.Build.Options["cache"], info.RunnerCommand, "cache", cacheFile)
	}

	// Find artifacts
	b.archiveFiles(w, info.Build.Options["artifacts"], info.RunnerCommand, "artifacts", "artifacts.tgz")

	// If archive is created upload it
	b.writeIfFile(w, "artifacts.tgz")
	b.echoColored(w, "Uploading artifacts...")
	b.executeCommand(w, "du", "-h", "artifacts.tgz")
	b.executeCommand(w, "curl", "-s", "-S", "--fail", "--retry", "3", "-X", "POST",
		"-#",
		"-o", "artifacts.upload.log",
		"-H", "BUILD-TOKEN: "+info.Build.Token,
		"-F", "file=@artifacts.tgz",
		info.Build.Network.GetArtifactsUploadURL(info.Build.Runner.RunnerCredentials, info.Build.ID))
	b.executeCommand(w, "rm", "-f", "artifacts.tgz")
	b.writeEndIf(w)

	w.Flush()

	return b.finalize(buffer.String())
}

func (b *BashShell) finalize(script string) string {
	var buffer bytes.Buffer
	w := bufio.NewWriter(&buffer)
	io.WriteString(w, "#!/usr/bin/env bash\n\n")
	io.WriteString(w, "set -eo pipefail\n")
	io.WriteString(w, ": | eval "+helpers.ShellEscape(script)+"\n")
	w.Flush()
	return buffer.String()
}

func (b *BashShell) GenerateScript(info common.ShellScriptInfo) (*common.ShellScript, error) {
	script := common.ShellScript{
		PreScript:   b.generatePreBuildScript(info),
		BuildScript: b.generateCommands(info),
		PostScript:  b.generatePostBuildScript(info),
		Environment: b.GetVariables(info),
	}

	// su
	if info.User != nil {
		script.Command = "su"
		if info.Type == common.LoginShell {
			script.Arguments = []string{"--shell", "/bin/bash", "--login", *info.User}
		} else {
			script.Arguments = []string{"--shell", "/bin/bash", *info.User}
		}
	} else {
		script.Command = "bash"
		if info.Type == common.LoginShell {
			script.Arguments = []string{"--login"}
		}
	}

	return &script, nil
}

func (b *BashShell) IsDefault() bool {
	return runtime.GOOS != "windows"
}

func init() {
	common.RegisterShell(&BashShell{
		AbstractShell: AbstractShell{
			SupportedOptions: []string{"artifacts", "cache"},
		},
	})
}
