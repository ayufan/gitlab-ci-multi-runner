package network

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	. "gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"io"
	"mime/multipart"
	"net/http"
	"runtime"
)

const clientError = -100

type GitLabClient struct {
	clients map[string]*client
}

func (n *GitLabClient) getClient(runner RunnerCredentials) (c *client, err error) {
	if n.clients == nil {
		n.clients = make(map[string]*client)
	}
	key := fmt.Sprintf("%s_%d_%s", runner.URL, runner.TLSSkipVerify, runner.TLSCAFile)
	c = n.clients[key]
	if c == nil {
		c, err = newClient(runner)
		if err != nil {
			return
		}
		n.clients[key] = c
	}
	return
}

func (n *GitLabClient) getRunnerVersion(config RunnerConfig) VersionInfo {
	info := VersionInfo{
		Name:         NAME,
		Version:      VERSION,
		Revision:     REVISION,
		Platform:     runtime.GOOS,
		Architecture: runtime.GOARCH,
		Executor:     config.Executor,
	}

	if executor := GetExecutor(config.Executor); executor != nil {
		executor.GetFeatures(&info.Features)
	}

	if config.Shell != nil {
		if shell := GetShell(*config.Shell); shell != nil {
			shell.GetFeatures(&info.Features)
		}
	}

	return info
}

func (n *GitLabClient) do(runner RunnerCredentials, uri string, prepRequest RequestPreparer, statusCode int, response interface{}) (int, string) {
	c, err := n.getClient(runner)
	if err != nil {
		return clientError, err.Error()
	}

	return c.do(uri, prepRequest, statusCode, response)
}

func (n *GitLabClient) doJson(runner RunnerCredentials, method, uri string, statusCode int, request interface{}, response interface{}) (int, string) {
	c, err := n.getClient(runner)
	if err != nil {
		return clientError, err.Error()
	}

	return c.doJson(uri, method, statusCode, request, response)
}

func (n *GitLabClient) GetBuild(config RunnerConfig) (*GetBuildResponse, bool) {
	request := GetBuildRequest{
		Info:  n.getRunnerVersion(config),
		Token: config.Token,
	}

	var response GetBuildResponse
	result, statusText := n.doJson(config.RunnerCredentials, "POST", "builds/register.json", 201, &request, &response)

	switch result {
	case 201:
		logrus.Println(config.ShortDescription(), "Checking for builds...", "received")
		return &response, true
	case 403:
		logrus.Errorln(config.ShortDescription(), "Checking for builds...", "forbidden")
		return nil, false
	case 404:
		logrus.Debugln(config.ShortDescription(), "Checking for builds...", "nothing")
		return nil, true
	case clientError:
		logrus.Errorln(config.ShortDescription(), "Checking for builds...", "error:", statusText)
		return nil, false
	default:
		logrus.Warningln(config.ShortDescription(), "Checking for builds...", "failed:", statusText)
		return nil, true
	}
}

func (n *GitLabClient) RegisterRunner(runner RunnerCredentials, description, tags string) *RegisterRunnerResponse {
	// TODO: pass executor
	request := RegisterRunnerRequest{
		Info:        n.getRunnerVersion(RunnerConfig{}),
		Token:       runner.Token,
		Description: description,
		Tags:        tags,
	}

	var response RegisterRunnerResponse
	result, statusText := n.doJson(runner, "POST", "runners/register.json", 201, &request, &response)
	shortToken := helpers.ShortenToken(runner.Token)

	switch result {
	case 201:
		logrus.Println(shortToken, "Registering runner...", "succeeded")
		return &response
	case 403:
		logrus.Errorln(shortToken, "Registering runner...", "forbidden (check registration token)")
		return nil
	case clientError:
		logrus.Errorln(shortToken, "Registering runner...", "error", statusText)
		return nil
	default:
		logrus.Errorln(shortToken, "Registering runner...", "failed", statusText)
		return nil
	}
}

func (n *GitLabClient) DeleteRunner(runner RunnerCredentials) bool {
	request := DeleteRunnerRequest{
		Token: runner.Token,
	}

	result, statusText := n.doJson(runner, "DELETE", "runners/delete", 200, &request, nil)
	shortToken := helpers.ShortenToken(runner.Token)

	switch result {
	case 200:
		logrus.Println(shortToken, "Deleting runner...", "succeeded")
		return true
	case 403:
		logrus.Errorln(shortToken, "Deleting runner...", "forbidden")
		return false
	case clientError:
		logrus.Errorln(shortToken, "Deleting runner...", "error", statusText)
		return false
	default:
		logrus.Errorln(shortToken, "Deleting runner...", "failed", statusText)
		return false
	}
}

func (n *GitLabClient) VerifyRunner(runner RunnerCredentials) bool {
	request := VerifyRunnerRequest{
		Token: runner.Token,
	}

	// HACK: we use non-existing build id to check if receive forbidden or not found
	result, statusText := n.doJson(runner, "PUT", fmt.Sprintf("builds/%d", -1), 200, &request, nil)
	shortToken := helpers.ShortenToken(runner.Token)

	switch result {
	case 404:
		// this is expected due to fact that we ask for non-existing job
		logrus.Println(shortToken, "Veryfing runner...", "is alive")
		return true
	case 403:
		logrus.Errorln(shortToken, "Veryfing runner...", "is removed")
		return false
	case clientError:
		logrus.Errorln(shortToken, "Veryfing runner...", "error", statusText)
		return false
	default:
		logrus.Errorln(shortToken, "Veryfing runner...", "failed", statusText)
		return true
	}
}

func (n *GitLabClient) UpdateBuild(config RunnerConfig, id int, state BuildState, trace string) UpdateState {
	request := UpdateBuildRequest{
		Info:  n.getRunnerVersion(config),
		Token: config.Token,
		State: state,
		Trace: trace,
	}

	result, statusText := n.doJson(config.RunnerCredentials, "PUT", fmt.Sprintf("builds/%d.json", id), 200, &request, nil)
	switch result {
	case 200:
		logrus.Println(config.ShortDescription(), id, "Submitting build to coordinator...", "ok")
		return UpdateSucceeded
	case 404:
		logrus.Warningln(config.ShortDescription(), id, "Submitting build to coordinator...", "aborted")
		return UpdateAbort
	case 403:
		logrus.Errorln(config.ShortDescription(), id, "Submitting build to coordinator...", "forbidden")
		return UpdateAbort
	case clientError:
		logrus.Errorln(config.ShortDescription(), id, "Submitting build to coordinator...", "error", statusText)
		return UpdateAbort
	default:
		logrus.Warningln(config.ShortDescription(), id, "Submitting build to coordinator...", "failed", statusText)
		return UpdateFailed
	}
}

func (n *GitLabClient) GetArtifactsUploadURL(config RunnerCredentials, id int) string {
	c, err := n.getClient(config)
	if err != nil {
		return ""
	}
	return c.fullUrl("builds/%d/artifacts", id)
}

func (n *GitLabClient) UploadArtifacts(config RunnerConfig, id int, data io.Reader) bool {
	result, statusText := n.do(config.RunnerCredentials, n.GetArtifactsUploadURL(config.RunnerCredentials, id), func(url string) (*http.Request, error) {
		pipeOut, pipeIn := io.Pipe()
		
		mpw := multipart.NewWriter(pipeIn)
		wr, err := mpw.CreateFormFile("file", "artifacts.tgz")
		if err != nil {
			return nil, err
		}
		
		if _, err := io.Copy(wr, data); err != nil {
			return nil, err
		}

		if err := mpw.Close(); err != nil {
			return nil, err
		}

		if err := pipeIn.Close(); err != nil {
			return nil, err
		}

		req, err := http.NewRequest("POST", url, pipeOut)
		if err != nil {
			return nil, err
		}
		
		req.Header.Set("Content-Type", mpw.FormDataContentType())
		req.Header.Set("BUILD-TOKEN", config.RunnerCredentials.Token)

		return req, nil
	}, 200, nil)

	switch result {
	case 200:
		logrus.Println(config.ShortDescription(), id, "Uploading artifacts to coordinator...", "ok")
		return true
	case 403:
		logrus.Errorln(config.ShortDescription(), id, "Uploading artifacts to coordinator...", "forbidden")
		return false
	case clientError:
		logrus.Errorln(config.ShortDescription(), id, "Uploading artifacts to coordinator...", "error", statusText)
		return false
	default:
		logrus.Warningln(config.ShortDescription(), id, "Uploading artifacts to coordinator...", "failed", statusText)
		return false
	}
}
