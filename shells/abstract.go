package shells

import (
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"path/filepath"
	"strconv"
	"strings"
)

type AbstractShell struct {
}

type ShellWriter interface {
	Variable(variable common.BuildVariable)
	Command(command string, arguments ...string)
	Line(text string)

	IfDirectory(path string)
	IfFile(file string)
	Else()
	EndIf()

	Cd(path string)
	RmDir(path string)
	RmFile(path string)
	Absolute(path string) string

	Print(fmt string, arguments ...interface{})
	Notice(fmt string, arguments ...interface{})
	Warning(fmt string, arguments ...interface{})
	Error(fmt string, arguments ...interface{})
	EmptyLine()
}

func (b *AbstractShell) GetFeatures(features *common.FeaturesInfo) {
	features.Artifacts = true
	features.Cache = true
}

func (s *AbstractShell) GetSupportedOptions() []string {
	return []string{"artifacts", "cache"}
}

func (b *AbstractShell) writeCdBuildDir(w ShellWriter, info common.ShellScriptInfo) {
	w.Cd(info.Build.FullProjectDir())
}

func (b *AbstractShell) writeExports(w ShellWriter, info common.ShellScriptInfo) {
	for _, variable := range info.Build.GetAllVariables() {
		w.Variable(variable)
	}
}

func (b *AbstractShell) writeCloneCmd(w ShellWriter, build *common.Build, projectDir string) {
	w.Notice("Cloning repository...")
	w.RmDir(projectDir)
	w.Command("git", "clone", build.RepoURL, projectDir)
	w.Cd(projectDir)
}

func (b *AbstractShell) writeFetchCmd(w ShellWriter, build *common.Build, projectDir string, gitDir string) {
	w.IfDirectory(gitDir)
	w.Notice("Fetching changes...")
	w.Cd(projectDir)
	w.Command("git", "clean", "-ffdx")
	w.Command("git", "reset", "--hard")
	w.Command("git", "remote", "set-url", "origin", build.RepoURL)
	w.Command("git", "fetch", "origin")
	w.Else()
	b.writeCloneCmd(w, build, projectDir)
	w.EndIf()
}

func (b *AbstractShell) writeCheckoutCmd(w ShellWriter, build *common.Build) {
	w.Notice("Checking out %s as %s...", build.Sha[0:8], build.RefName)
	w.Command("git", "checkout", build.Sha)
}

func (b *AbstractShell) GeneratePreBuild(w ShellWriter, info common.ShellScriptInfo) {
	b.writeExports(w, info)

	build := info.Build
	projectDir := build.FullProjectDir()
	gitDir := filepath.Join(build.FullProjectDir(), ".git")

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
		w.IfFile(cacheFile)
		b.extractFiles(w, info.RunnerCommand, "cache", cacheFile)
		if cacheFile2 != "" {
			w.Else()

			// If we have cache, restore it
			w.IfFile(cacheFile2)
			b.extractFiles(w, info.RunnerCommand, "cache", cacheFile2)
			w.EndIf()
		}
		w.EndIf()
	}
}

func (b *AbstractShell) GenerateCommands(w ShellWriter, info common.ShellScriptInfo) {
	b.writeExports(w, info)
	b.writeCdBuildDir(w, info)

	commands := info.Build.Commands
	commands = strings.TrimSpace(commands)
	for _, command := range strings.Split(commands, "\n") {
		command = strings.TrimSpace(command)
		if !helpers.BoolOrDefault(info.Build.Runner.DisableVerbose, false) {
			if command != "" {
				w.Notice("$ %s", command)
			} else {
				w.EmptyLine()
			}
		}
		w.Line(command)
	}
}

func (b *AbstractShell) archiveFiles(w ShellWriter, list interface{}, runnerCommand, archiveType, archivePath string) {
	hash, ok := helpers.ToConfigMap(list)
	if !ok {
		return
	}

	if runnerCommand == "" {
		w.Warning("The %s is not supported in this executor.", archiveType)
		return
	}

	args := []string{
		"archive",
		"--file",
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
	w.Notice("Archiving %s...", archiveType)
	w.Command(runnerCommand, args...)
}

func (b *AbstractShell) extractFiles(w ShellWriter, runnerCommand, archiveType, archivePath string) {
	if runnerCommand == "" {
		w.Warning("The %s is not supported in this executor.", archiveType)
		return
	}

	args := []string{
		"extract",
		"--file",
		archivePath,
	}

	// Execute extract command
	w.Notice("Restoring %s...", archiveType)
	w.Command(runnerCommand, args...)
}

func (b *AbstractShell) uploadArtifacts(w ShellWriter, build *common.Build, runnerCommand, archivePath string) {
	if runnerCommand == "" {
		w.Warning("The artifacts uploading is not supported in this executor.")
		return
	}

	args := []string{
		"artifacts",
		"--url",
		build.Runner.URL,
		"--token",
		build.Token,
		"--id",
		strconv.Itoa(build.ID),
		"--file",
		archivePath,
	}

	w.Notice("Uploading artifacts...")
	w.Command(runnerCommand, args...)
}

func (b *AbstractShell) GeneratePostBuild(w ShellWriter, info common.ShellScriptInfo) {
	b.writeExports(w, info)
	b.writeCdBuildDir(w, info)

	// Find cached files and archive them
	if cacheFile := info.Build.CacheFile(); cacheFile != "" {
		b.archiveFiles(w, info.Build.Options["cache"], info.RunnerCommand, "cache", cacheFile)
	}

	if info.Build.Network != nil {
		// Find artifacts
		b.archiveFiles(w, info.Build.Options["artifacts"], info.RunnerCommand, "artifacts", "artifacts.zip")

		// If archive is created upload it
		w.IfFile("artifacts.zip")
		b.uploadArtifacts(w, info.Build, info.RunnerCommand, "artifacts.zip")
		w.RmFile("aritfacts.zip")
		w.EndIf()
	}
}
