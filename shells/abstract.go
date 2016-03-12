package shells

import (
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
)

type AbstractShell struct {
}

func (b *AbstractShell) GetFeatures(features *common.FeaturesInfo) {
	features.Artifacts = true
	features.Cache = true
}

func (b *AbstractShell) GetSupportedOptions() []string {
	return []string{"artifacts", "cache", "dependencies"}
}

func (b *AbstractShell) writeCdBuildDir(w ShellWriter, info common.ShellScriptInfo) {
	w.Cd(info.Build.FullProjectDir())
}

func (b *AbstractShell) writeExports(w ShellWriter, info common.ShellScriptInfo) {
	for _, variable := range info.Build.GetAllVariables() {
		w.Variable(variable)
	}
}

func (b *AbstractShell) writeTLSCAInfo(w ShellWriter, build *common.Build, key string) {
	if build.TLSCAChain != "" {
		w.Variable(common.BuildVariable{
			Key:      key,
			Value:    build.TLSCAChain,
			Public:   true,
			Internal: true,
			File:     true,
		})
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
	w.Command("git", "fetch", "origin", "--prune", "+refs/heads/*:refs/remotes/origin/*", "+refs/tags/*:refs/tags/*")
	w.Else()
	b.writeCloneCmd(w, build, projectDir)
	w.EndIf()
}

func (b *AbstractShell) writeCheckoutCmd(w ShellWriter, build *common.Build) {
	w.Notice("Checking out %s as %s...", build.Sha[0:8], build.RefName)
	// We remove a git index file, this is required if `git checkout` is terminated
	w.RmFile(".git/index.lock")
	w.Command("git", "checkout", build.Sha)
}

func (b *AbstractShell) cacheFile(cacheKey string, info common.ShellScriptInfo) string {
	if cacheKey == "" {
		return ""
	}
	cacheFile := path.Join(info.Build.CacheDir, cacheKey)
	cacheFile, err := filepath.Rel(info.Build.BuildDir, cacheFile)
	if err != nil {
		return ""
	}
	return cacheFile
}

func (b *AbstractShell) encodeArchiverOptions(list interface{}) (args []string) {
	hash, ok := helpers.ToConfigMap(list)
	if !ok {
		return
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
	return
}

func (b *AbstractShell) cacheExtractor(w ShellWriter, list interface{}, info common.ShellScriptInfo, cacheKey string) {
	if info.RunnerCommand == "" {
		w.Warning("The cache is not supported in this executor.")
		return
	}

	// Create list of files to archive
	archiverArgs := b.encodeArchiverOptions(list)
	if len(archiverArgs) == 0 {
		// Skip restoring cache if no cache is defined
		return
	}

	args := []string{
		"cache-extractor",
		"--file", b.cacheFile(cacheKey, info),
	}

	// Generate cache download address
	if url := getCacheDownloadURL(info.Build, cacheKey); url != "" {
		args = append(args, "--url", url)
	}

	// Execute archive command
	w.Notice("Checking cache for %s...", cacheKey)
	w.Command(info.RunnerCommand, args...)
}

func (b *AbstractShell) downloadArtifacts(w ShellWriter, build *common.BuildInfo, info common.ShellScriptInfo) {
	if info.RunnerCommand == "" {
		w.Warning("The artifacts downloading is not supported in this executor.")
		return
	}

	args := []string{
		"artifacts-downloader",
		"--url",
		info.Build.Runner.URL,
		"--token",
		build.Token,
		"--id",
		strconv.Itoa(build.ID),
	}

	w.Notice("Downloading artifacts for %s (%d)...", build.Name, build.ID)
	w.Command(info.RunnerCommand, args...)
}

func (b *AbstractShell) isDependentBuild(build *common.Build, name string) bool {
	dependencies, ok := build.Options["dependencies"].([]interface{})
	if !ok {
		// If no dependencies are defined we assume that we depend on all builds
		return true
	}

	for _, dependency := range dependencies {
		if value, ok := dependency.(string); ok && name == value {
			return true
		}
	}
	return false
}

func (b *AbstractShell) GeneratePreBuild(w ShellWriter, info common.ShellScriptInfo) {
	b.writeExports(w, info)

	build := info.Build
	projectDir := build.FullProjectDir()
	gitDir := path.Join(build.FullProjectDir(), ".git")

	b.writeTLSCAInfo(w, info.Build, "GIT_SSL_CAINFO")
	b.writeTLSCAInfo(w, info.Build, "CI_SERVER_TLS_CA_FILE")

	if build.AllowGitFetch {
		b.writeFetchCmd(w, build, projectDir, gitDir)
	} else {
		b.writeCloneCmd(w, build, projectDir)
	}

	b.writeCheckoutCmd(w, build)

	// Try to restore from main cache, if not found cache for master
	if cacheKey := info.Build.CacheKey(); cacheKey != "" {
		b.cacheExtractor(w, info.Build.Options["cache"], info, cacheKey)
	}

	// Process all artifacts
	for _, otherBuild := range info.Build.DependsOnBuilds {
		if otherBuild.Artifacts == nil || otherBuild.Artifacts.Filename == "" {
			continue
		}
		if !b.isDependentBuild(info.Build, otherBuild.Name) {
			continue
		}
		b.downloadArtifacts(w, &otherBuild, info)
	}
}

func (b *AbstractShell) GenerateCommands(w ShellWriter, info common.ShellScriptInfo) {
	b.writeExports(w, info)
	b.writeCdBuildDir(w, info)

	commands := info.Build.Commands
	commands = strings.TrimSpace(commands)
	for _, command := range strings.Split(commands, "\n") {
		command = strings.TrimSpace(command)
		if command != "" {
			w.Notice("$ %s", command)
		} else {
			w.EmptyLine()
		}
		w.Line(command)
	}
}

func (b *AbstractShell) cacheArchiver(w ShellWriter, list interface{}, info common.ShellScriptInfo, cacheKey string) {
	if info.RunnerCommand == "" {
		w.Warning("The cache is not supported in this executor.")
		return
	}

	args := []string{
		"cache-archiver",
		"--file", b.cacheFile(cacheKey, info),
	}

	// Create list of files to archive
	archiverArgs := b.encodeArchiverOptions(list)
	if len(archiverArgs) == 0 {
		// Skip creating archive
		return
	}
	args = append(args, archiverArgs...)

	// Generate cache upload address
	if url := getCacheUploadURL(info.Build, cacheKey); url != "" {
		args = append(args, "--url", url)
	}

	// Execute archive command
	w.Notice("Creating cache %s...", cacheKey)
	w.Command(info.RunnerCommand, args...)
}

func (b *AbstractShell) uploadArtifacts(w ShellWriter, list interface{}, info common.ShellScriptInfo) {
	if info.Build.Runner.URL == "" {
		return
	}
	if info.RunnerCommand == "" {
		w.Warning("The artifacts uploading is not supported in this executor.")
		return
	}

	args := []string{
		"artifacts-uploader",
		"--url",
		info.Build.Runner.URL,
		"--token",
		info.Build.Token,
		"--id",
		strconv.Itoa(info.Build.ID),
	}

	// Create list of files to archive
	archiverArgs := b.encodeArchiverOptions(list)
	if len(archiverArgs) == 0 {
		// Skip creating archive
		return
	}
	args = append(args, archiverArgs...)

	// Get artifacts:name
	if name, ok := helpers.GetMapKey(info.Build.Options["artifacts"], "name"); ok {
		if nameValue, ok := name.(string); ok && nameValue != "" {
			args = append(args, "--name", nameValue)
		}
	}

	w.Notice("Uploading artifacts...")
	w.Command(info.RunnerCommand, args...)
}

func (b *AbstractShell) GeneratePostBuild(w ShellWriter, info common.ShellScriptInfo) {
	b.writeExports(w, info)
	b.writeCdBuildDir(w, info)
	b.writeTLSCAInfo(w, info.Build, "CI_SERVER_TLS_CA_FILE")

	// Find cached files and archive them
	if cacheKey := info.Build.CacheKey(); cacheKey != "" {
		b.cacheArchiver(w, info.Build.Options["cache"], info, cacheKey)
	}

	// Upload artifacts
	b.uploadArtifacts(w, info.Build.Options["artifacts"], info)
}
