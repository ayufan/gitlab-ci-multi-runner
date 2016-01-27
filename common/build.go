package common

import (
	"bytes"
	"errors"
	"fmt"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type BuildState string

const (
	Pending BuildState = "pending"
	Running            = "running"
	Failed             = "failed"
	Success            = "success"
)

type Build struct {
	GetBuildResponse `yaml:",inline"`
	Network          Network
	BuildState       BuildState     `json:"build_state"`
	BuildStarted     time.Time      `json:"build_started"`
	BuildFinished    time.Time      `json:"build_finished"`
	BuildDuration    time.Duration  `json:"build_duration"`
	BuildAbort       chan os.Signal `json:"-" yaml:"-"`
	RootDir          string         `json:"-" yaml:"-"`
	BuildDir         string         `json:"-" yaml:"-"`
	CacheDir         string         `json:"-" yaml:"-"`
	Hostname         string         `json:"-" yaml:"-"`
	Runner           *RunnerConfig  `json:"runner"`

	// Unique ID for all running builds (globally)
	GlobalID int `json:"global_id"`

	// Unique ID for all running builds on this runner
	RunnerID int `json:"runner_id"`

	// Unique ID for all running builds on this runner and this project
	ProjectRunnerID int `json:"project_runner_id"`

	buildLog     bytes.Buffer
	buildLogLock sync.RWMutex
}

func (b *Build) AssignID(otherBuilds ...*Build) {
	globals := make(map[int]bool)
	runners := make(map[int]bool)
	projectRunners := make(map[int]bool)

	for _, otherBuild := range otherBuilds {
		globals[otherBuild.GlobalID] = true

		if otherBuild.Runner.ShortDescription() != b.Runner.ShortDescription() {
			continue
		}
		runners[otherBuild.RunnerID] = true

		if otherBuild.ProjectID != b.ProjectID {
			continue
		}
		projectRunners[otherBuild.ProjectRunnerID] = true
	}

	for i := 0; ; i++ {
		if !globals[i] {
			b.GlobalID = i
			break
		}
	}

	for i := 0; ; i++ {
		if !runners[i] {
			b.RunnerID = i
			break
		}
	}

	for i := 0; ; i++ {
		if !projectRunners[i] {
			b.ProjectRunnerID = i
			break
		}
	}
}

func (b *Build) ProjectUniqueName() string {
	return fmt.Sprintf("runner-%s-project-%d-concurrent-%d",
		b.Runner.ShortDescription(), b.ProjectID, b.ProjectRunnerID)
}

func (b *Build) ProjectSlug() (string, error) {
	url, err := url.Parse(b.RepoURL)
	if err != nil {
		return "", err
	}
	if url.Host == "" {
		return "", errors.New("only URI reference supported")
	}

	slug := url.Path
	slug = strings.TrimSuffix(slug, ".git")
	slug = path.Clean(slug)
	if slug == "." {
		return "", errors.New("invalid path")
	}
	if strings.Contains(slug, "..") {
		return "", errors.New("it doesn't look like a valid path")
	}
	return slug, nil
}

func (b *Build) ProjectUniqueDir(sharedDir bool) string {
	dir, err := b.ProjectSlug()
	if err != nil {
		dir = fmt.Sprintf("project-%d", b.ProjectID)
	}

	// for shared dirs path is constructed like this:
	// <some-path>/runner-short-id/concurrent-id/group-name/project-name/
	// ex.<some-path>/01234567/0/group/repo/
	if sharedDir {
		dir = path.Join(
			fmt.Sprintf("%s", b.Runner.ShortDescription()),
			fmt.Sprintf("%d", b.ProjectRunnerID),
			dir,
		)
	}
	return dir
}

func (b *Build) FullProjectDir() string {
	return helpers.ToSlash(b.BuildDir)
}

func (b *Build) CacheFileForRef(ref string) string {
	if b.CacheDir != "" {
		cacheKey := path.Join(b.Name, ref)

		// Get cache:key
		if hash, ok := b.Options["cache"].(map[string]interface{}); ok {
			if key, ok := hash["key"].(string); ok && key != "" {
				cacheKey = key
			}
		}

		// Ignore groups that are nil
		if cacheKey == "" {
			return ""
		}

		cacheFile := path.Join(b.CacheDir, cacheKey, "cache.zip")
		cacheFile, err := filepath.Rel(b.BuildDir, cacheFile)
		if err != nil {
			return ""
		}
		return filepath.ToSlash(cacheFile)
	}
	return ""
}

func (b *Build) CacheFile() string {
	// For tags we don't create cache
	if b.Tag {
		return ""
	}
	return b.CacheFileForRef(b.RefName)
}

func (b *Build) StartBuild(rootDir, cacheDir string, sharedDir bool) {
	b.BuildStarted = time.Now()
	b.BuildState = Pending
	b.RootDir = rootDir
	b.BuildDir = path.Join(rootDir, b.ProjectUniqueDir(sharedDir))
	b.CacheDir = path.Join(cacheDir, b.ProjectUniqueDir(false))
}

func (b *Build) FinishBuild(buildState BuildState) {
	b.BuildState = buildState
	b.BuildFinished = time.Now()
	b.BuildDuration = b.BuildFinished.Sub(b.BuildStarted)
}

func (b *Build) BuildLog() string {
	b.buildLogLock.RLock()
	defer b.buildLogLock.RUnlock()
	return b.buildLog.String()
}

func (b *Build) BuildLogLen() int {
	b.buildLogLock.RLock()
	defer b.buildLogLock.RUnlock()
	return b.buildLog.Len()
}

func (b *Build) WriteString(data string) (int, error) {
	b.buildLogLock.Lock()
	defer b.buildLogLock.Unlock()
	return b.buildLog.WriteString(data)
}

func (b *Build) WriteRune(r rune) (int, error) {
	b.buildLogLock.Lock()
	defer b.buildLogLock.Unlock()
	return b.buildLog.WriteRune(r)
}

func (b *Build) SendBuildLog() {
	var buildTrace string

	if b.Network == nil {
		return
	}

	buildTrace = b.BuildLog()
	for {
		if b.Network.UpdateBuild(*b.Runner, b.ID, b.BuildState, buildTrace) != UpdateFailed {
			break
		} else {
			time.Sleep(UpdateRetryInterval * time.Second)
		}
	}
}

func (b *Build) Run(globalConfig *Config) error {
	executor := NewExecutor(b.Runner.Executor)
	if executor == nil {
		b.WriteString("Executor not found: " + b.Runner.Executor)
		b.SendBuildLog()
		return errors.New("executor not found")
	}

	err := executor.Prepare(globalConfig, b.Runner, b)
	if err == nil {
		err = executor.Start()
	}
	if err == nil {
		err = executor.Wait()
	}
	executor.Finish(err)
	if executor != nil {
		executor.Cleanup()
	}
	return err
}

func (b *Build) String() string {
	return helpers.ToYAML(b)
}

func (b *Build) GetDefaultVariables() BuildVariables {
	return BuildVariables{
		{"CI", "true", true, true, false},
		{"CI_BUILD_REF", b.Sha, true, true, false},
		{"CI_BUILD_BEFORE_SHA", b.BeforeSha, true, true, false},
		{"CI_BUILD_REF_NAME", b.RefName, true, true, false},
		{"CI_BUILD_ID", strconv.Itoa(b.ID), true, true, false},
		{"CI_BUILD_REPO", b.RepoURL, true, true, false},
		{"CI_PROJECT_ID", strconv.Itoa(b.ProjectID), true, true, false},
		{"CI_PROJECT_DIR", b.FullProjectDir(), true, true, false},
		{"CI_SERVER", "yes", true, true, false},
		{"CI_SERVER_NAME", "GitLab CI", true, true, false},
		{"CI_SERVER_VERSION", "", true, true, false},
		{"CI_SERVER_REVISION", "", true, true, false},
		{"GITLAB_CI", "true", true, true, false},
	}
}

func (b *Build) GetAllVariables() BuildVariables {
	variables := b.Runner.GetVariables()
	variables = append(variables, b.GetDefaultVariables()...)
	variables = append(variables, b.Variables...)
	return variables.Expand()
}
