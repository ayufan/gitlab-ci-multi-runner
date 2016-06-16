package common

import (
	"io"
)

type UpdateState int
type UploadState int
type DownloadState int

const (
	UpdateSucceeded UpdateState = iota
	UpdateNotFound
	UpdateAbort
	UpdateFailed
	UpdateRangeMissmatch
)

const (
	UploadSucceeded UploadState = iota
	UploadTooLarge
	UploadForbidden
	UploadFailed
)

const (
	DownloadSucceeded DownloadState = iota
	DownloadForbidden
	DownloadFailed
	DownloadNotFound
)

type FeaturesInfo struct {
	Variables bool `json:"variables"`
	Image     bool `json:"image"`
	Services  bool `json:"services"`
	Artifacts bool `json:"features"`
	Cache     bool `json:"cache"`
}

type VersionInfo struct {
	Name         string       `json:"name,omitempty"`
	Version      string       `json:"version,omitempty"`
	Revision     string       `json:"revision,omitempty"`
	Platform     string       `json:"platform,omitempty"`
	Architecture string       `json:"architecture,omitempty"`
	Executor     string       `json:"executor,omitempty"`
	Features     FeaturesInfo `json:"features"`
}

type GetBuildRequest struct {
	Info  VersionInfo `json:"info,omitempty"`
	Token string      `json:"token,omitempty"`
}

type BuildArtifacts struct {
	Filename string `json:"filename,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

type BuildInfo struct {
	ID        int             `json:"id,omitempty"`
	Sha       string          `json:"sha,omitempty"`
	RefName   string          `json:"ref,omitempty"`
	Token     string          `json:"token"`
	Name      string          `json:"name"`
	Stage     string          `json:"stage"`
	Tag       bool            `json:"tag"`
	Artifacts *BuildArtifacts `json:"artifacts_file"`
}

type GetBuildResponse struct {
	ID              int            `json:"id,omitempty"`
	ProjectID       int            `json:"project_id,omitempty"`
	Commands        string         `json:"commands,omitempty"`
	RepoURL         string         `json:"repo_url,omitempty"`
	Sha             string         `json:"sha,omitempty"`
	RefName         string         `json:"ref,omitempty"`
	BeforeSha       string         `json:"before_sha,omitempty"`
	AllowGitFetch   bool           `json:"allow_git_fetch,omitempty"`
	Timeout         int            `json:"timeout,omitempty"`
	Variables       BuildVariables `json:"variables"`
	Options         BuildOptions   `json:"options"`
	Token           string         `json:"token"`
	Name            string         `json:"name"`
	Stage           string         `json:"stage"`
	Tag             bool           `json:"tag"`
	DependsOnBuilds []BuildInfo    `json:"depends_on_builds"`
	TLSCAChain      string         `json:"-"`
}

type RegisterRunnerRequest struct {
	Info        VersionInfo `json:"info,omitempty"`
	Token       string      `json:"token,omitempty"`
	Description string      `json:"description,omitempty"`
	Tags        string      `json:"tag_list,omitempty"`
}

type RegisterRunnerResponse struct {
	Token string `json:"token,omitempty"`
}

type DeleteRunnerRequest struct {
	Token string `json:"token,omitempty"`
}

type VerifyRunnerRequest struct {
	Token string `json:"token,omitempty"`
}

type UpdateBuildRequest struct {
	Info  VersionInfo `json:"info,omitempty"`
	Token string      `json:"token,omitempty"`
	State BuildState  `json:"state,omitempty"`
	Trace *string     `json:"trace,omitempty"`
}

type BuildCredentials struct {
	ID        int    `long:"id" env:"CI_BUILD_ID" description:"The build ID to upload artifacts for"`
	Token     string `long:"token" env:"CI_BUILD_TOKEN" required:"true" description:"Build token"`
	URL       string `long:"url" env:"CI_SERVER_URL" required:"true" description:"GitLab CI URL"`
	TLSCAFile string `long:"tls-ca-file" env:"CI_SERVER_TLS_CA_FILE" description:"File containing the certificates to verify the peer when using HTTPS"`
}

type BuildTrace interface {
	io.Writer
	Success()
	Fail(err error)
	Notify(abort func())
	IsStdout() bool
}

type BuildTracePatch interface {
	Patch() []byte
	Offset() int
	Limit() int
	SetNewOffset(newOffset int)
}

type Network interface {
	GetBuild(config RunnerConfig) (*GetBuildResponse, bool)
	RegisterRunner(config RunnerCredentials, description, tags string) *RegisterRunnerResponse
	DeleteRunner(config RunnerCredentials) bool
	VerifyRunner(config RunnerCredentials) bool
	UpdateBuild(config RunnerConfig, id int, state BuildState, trace *string) UpdateState
	PatchTrace(config RunnerConfig, buildCredentials *BuildCredentials, tracePart BuildTracePatch) UpdateState
	DownloadArtifacts(config BuildCredentials, artifactsFile string) DownloadState
	UploadRawArtifacts(config BuildCredentials, reader io.Reader, baseName string, expireIn string) UploadState
	UploadArtifacts(config BuildCredentials, artifactsFile string) UploadState
	ProcessBuild(config RunnerConfig, buildCredentials *BuildCredentials) BuildTrace
}
