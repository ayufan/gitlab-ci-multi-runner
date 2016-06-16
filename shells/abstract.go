package shells

import (
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"errors"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

type AbstractShell struct {
}

func (b *AbstractShell) GetFeatures(features *common.FeaturesInfo) {
	features.Artifacts = true
	features.Cache = true
}

func (b *AbstractShell) GetSupportedOptions() []string {
	return []string{"artifacts", "cache", "dependencies", "after_script"}
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
	w.RmDir(projectDir)
	if depth := build.GetGitDepth(); depth != "" {
		w.Notice("Cloning repository for %s with git depth set to %s...", build.RefName, depth)
		w.Command("git", "clone", build.RepoURL, projectDir, "--depth", depth, "--branch", build.RefName)
	} else {
		w.Notice("Cloning repository...")
		w.Command("git", "clone", build.RepoURL, projectDir)
	}
	w.Cd(projectDir)
}

func (b *AbstractShell) writeFetchCmd(w ShellWriter, build *common.Build, projectDir string, gitDir string) {
	depth := build.GetGitDepth()

	w.IfDirectory(gitDir)
	if depth != "" {
		w.Notice("Fetching changes for %s with git depth set to %s...", build.RefName, depth)
	} else {
		w.Notice("Fetching changes...")
	}
	w.Cd(projectDir)
	w.Command("git", "clean", "-ffdx")
	w.Command("git", "reset", "--hard")
	w.Command("git", "remote", "set-url", "origin", build.RepoURL)
	if depth != "" {
		var refspec string
		if build.Tag {
			refspec = "+refs/tags/" + build.RefName + ":refs/tags/" + build.RefName
		} else {
			refspec = "+refs/heads/" + build.RefName + ":refs/remotes/origin/" + build.RefName
		}
		w.Command("git", "fetch", "--depth", depth, "origin", "--prune", refspec)
	} else {
		w.Command("git", "fetch", "origin", "--prune", "+refs/heads/*:refs/remotes/origin/*", "+refs/tags/*:refs/tags/*")
	}
	w.Else()
	b.writeCloneCmd(w, build, projectDir)
	w.EndIf()
}

func (b *AbstractShell) writeCheckoutCmd(w ShellWriter, build *common.Build) {
	w.Notice("Checking out %s as %s...", build.Sha[0:8], build.RefName)
	// We remove a git index file, this is required if `git checkout` is terminated
	w.RmFile(".git/index.lock")
	w.Command("git", "checkout", "-q", build.Sha)
}

func (b *AbstractShell) cacheFile(build *common.Build, userKey string) (key, file string) {
	if build.CacheDir == "" {
		return
	}

	// Deduce cache key
	key = path.Join(build.Name, build.RefName)
	if userKey != "" {
		key = build.GetAllVariables().ExpandValue(userKey)
	}

	// Ignore cache without the key
	if key == "" {
		return
	}

	file = path.Join(build.CacheDir, key, "cache.zip")
	file, err := filepath.Rel(build.BuildDir, file)
	if err != nil {
		return "", ""
	}
	return
}

func (o *archivingOptions) CommandArguments() (args []string) {
	for _, path := range o.Paths {
		args = append(args, "--path", path)
	}

	if o.Untracked {
		args = append(args, "--untracked")
	}
	return
}

func (b *AbstractShell) cacheExtractor(w ShellWriter, options *archivingOptions, info common.ShellScriptInfo) {
	if options == nil {
		return
	}
	if info.RunnerCommand == "" {
		w.Warning("The cache is not supported in this executor.")
		return
	}

	// Skip restoring cache if no cache is defined
	if archiverArgs := options.CommandArguments(); len(archiverArgs) == 0 {
		return
	}

	// Skip archiving if no cache is defined
	cacheKey, cacheFile := b.cacheFile(info.Build, options.Key)
	if cacheKey == "" {
		return
	}

	args := []string{
		"cache-extractor",
		"--file", cacheFile,
	}

	// Generate cache download address
	if url := getCacheDownloadURL(info.Build, cacheKey); url != nil {
		args = append(args, "--url", url.String())
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

func (b *AbstractShell) downloadAllArtifacts(w ShellWriter, dependencies *dependencies, info common.ShellScriptInfo) {
	for _, otherBuild := range info.Build.DependsOnBuilds {
		if otherBuild.Artifacts == nil || otherBuild.Artifacts.Filename == "" {
			continue
		}
		if !dependencies.IsDependent(otherBuild.Name) {
			continue
		}
		b.downloadArtifacts(w, &otherBuild, info)
	}
}

func (b *AbstractShell) writePrepareScript(w ShellWriter, info common.ShellScriptInfo) (err error) {
	b.writeExports(w, info)

	build := info.Build
	projectDir := build.FullProjectDir()
	gitDir := path.Join(build.FullProjectDir(), ".git")

	b.writeTLSCAInfo(w, info.Build, "GIT_SSL_CAINFO")
	b.writeTLSCAInfo(w, info.Build, "CI_SERVER_TLS_CA_FILE")

	w.Command("git", "config", "--global", "fetch.recurseSubmodules", "false")
	if build.AllowGitFetch {
		b.writeFetchCmd(w, build, projectDir, gitDir)
	} else {
		b.writeCloneCmd(w, build, projectDir)
	}

	b.writeCheckoutCmd(w, build)

	// Parse options
	var options shellOptions
	err = info.Build.Options.Decode(&options)
	if err != nil {
		return
	}

	// Try to restore from main cache, if not found cache for master
	b.cacheExtractor(w, options.Cache, info)

	// Process all artifacts
	b.downloadAllArtifacts(w, options.Dependencies, info)
	return nil
}

func (b *AbstractShell) writeBuildScript(w ShellWriter, info common.ShellScriptInfo) (err error) {
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
		w.CheckForErrors()
	}

	return nil
}

func (b *AbstractShell) cacheArchiver(w ShellWriter, options *archivingOptions, info common.ShellScriptInfo) {
	if options == nil {
		return
	}
	if info.RunnerCommand == "" {
		w.Warning("The cache is not supported in this executor.")
		return
	}

	// Skip archiving if no cache is defined
	cacheKey, cacheFile := b.cacheFile(info.Build, options.Key)
	if cacheKey == "" {
		return
	}

	args := []string{
		"cache-archiver",
		"--file", cacheFile,
	}

	// Create list of files to archive
	archiverArgs := options.CommandArguments()
	if len(archiverArgs) == 0 {
		// Skip creating archive
		return
	}
	args = append(args, archiverArgs...)

	// Generate cache upload address
	if url := getCacheUploadURL(info.Build, cacheKey); url != nil {
		args = append(args, "--url", url.String())
	}

	// Execute archive command
	w.Notice("Creating cache %s...", cacheKey)
	w.Command(info.RunnerCommand, args...)
}

func (b *AbstractShell) uploadArtifacts(w ShellWriter, options *archivingOptions, info common.ShellScriptInfo) {
	if options == nil {
		return
	}
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
	archiverArgs := options.CommandArguments()
	if len(archiverArgs) == 0 {
		// Skip creating archive
		return
	}
	args = append(args, archiverArgs...)

	// Get artifacts:name
	if name, ok := info.Build.Options.GetString("artifacts", "name"); ok && name != "" {
		args = append(args, "--name", name)
	}

	// Get artifacts:expire_in
	if expireIn, ok := info.Build.Options.GetString("artifacts", "expire_in"); ok && expireIn != "" {
		args = append(args, "--expire-in", expireIn)
	}

	w.Notice("Uploading artifacts...")
	w.Command(info.RunnerCommand, args...)
}

func (b *AbstractShell) writeAfterScript(w ShellWriter, info common.ShellScriptInfo) error {
	shellOptions := struct {
		AfterScript []string `json:"after_script"`
	}{}
	err := info.Build.Options.Decode(&shellOptions)
	if err != nil {
		return err
	}

	if len(shellOptions.AfterScript) == 0 {
		return nil
	}

	b.writeExports(w, info)
	b.writeCdBuildDir(w, info)

	w.Notice("Running after script...")

	for _, command := range shellOptions.AfterScript {
		command = strings.TrimSpace(command)
		if command != "" {
			w.Notice("$ %s", command)
		} else {
			w.EmptyLine()
		}
		w.Line(command)
		w.CheckForErrors()
	}

	return nil
}

func (b *AbstractShell) writeArchiveCacheScript(w ShellWriter, info common.ShellScriptInfo) (err error) {
	// Parse options
	var options shellOptions
	err = info.Build.Options.Decode(&options)
	if err != nil {
		return
	}

	b.writeExports(w, info)
	b.writeCdBuildDir(w, info)
	b.writeTLSCAInfo(w, info.Build, "CI_SERVER_TLS_CA_FILE")

	// Find cached files and archive them
	b.cacheArchiver(w, options.Cache, info)
	return
}

func (b *AbstractShell) writeUploadArtifactsScript(w ShellWriter, info common.ShellScriptInfo) (err error) {
	// Parse options
	var options shellOptions
	err = info.Build.Options.Decode(&options)
	if err != nil {
		return
	}

	b.writeExports(w, info)
	b.writeCdBuildDir(w, info)
	b.writeTLSCAInfo(w, info.Build, "CI_SERVER_TLS_CA_FILE")

	// Upload artifacts
	b.uploadArtifacts(w, options.Artifacts, info)
	return
}

func (b *AbstractShell) writeScript(w ShellWriter, scriptType common.ShellScriptType, info common.ShellScriptInfo) (err error) {
	switch scriptType {
	case common.ShellPrepareScript:
		return b.writePrepareScript(w, info)

	case common.ShellBuildScript:
		return b.writeBuildScript(w, info)

	case common.ShellAfterScript:
		return b.writeAfterScript(w, info)

	case common.ShellArchiveCache:
		return b.writeArchiveCacheScript(w, info)

	case common.ShellUploadArtifacts:
		return b.writeUploadArtifactsScript(w, info)

	default:
		return errors.New("Not supported script type: " + string(scriptType))
	}
}
