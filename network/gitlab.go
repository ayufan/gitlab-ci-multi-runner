package network

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	. "gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"runtime"
)

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

func (n *GitLabClient) do(runner RunnerCredentials, uri, method string, statusCode int, request interface{}, response interface{}) (int, string) {
	c, err := n.getClient(runner)
	if err != nil {
		return -1, err.Error()
	}

	return c.do(uri, method, statusCode, request, response)
}

func (n *GitLabClient) post(runner RunnerCredentials, uri string, statusCode int, request interface{}, response interface{}) (int, string) {
	return n.do(runner, uri, "POST", statusCode, request, response)
}

func (n *GitLabClient) put(runner RunnerCredentials, uri string, statusCode int, request interface{}, response interface{}) (int, string) {
	return n.do(runner, uri, "PUT", statusCode, request, response)
}

func (n *GitLabClient) delete(runner RunnerCredentials, uri string, statusCode int, request interface{}, response interface{}) (int, string) {
	return n.do(runner, uri, "DELETE", statusCode, request, response)
}

func (n *GitLabClient) GetBuild(config RunnerConfig) (*GetBuildResponse, bool) {
	request := GetBuildRequest{
		Info:  n.getRunnerVersion(config),
		Token: config.Token,
	}

	var response GetBuildResponse
	result, statusText := n.post(config.RunnerCredentials, "builds/register.json", 201, &request, &response)

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
	result, statusText := n.post(runner, "runners/register.json", 201, &request, &response)
	shortToken := helpers.ShortenToken(runner.Token)

	switch result {
	case 201:
		logrus.Println(shortToken, "Registering runner...", "succeeded")
		return &response
	case 403:
		logrus.Errorln(shortToken, "Registering runner...", "forbidden (check registration token)")
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

	result, statusText := n.delete(runner, "runners/delete", 200, &request, nil)
	shortToken := helpers.ShortenToken(runner.Token)

	switch result {
	case 200:
		logrus.Println(shortToken, "Deleting runner...", "succeeded")
		return true
	case 403:
		logrus.Errorln(shortToken, "Deleting runner...", "forbidden")
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
	result, statusText := n.put(runner, fmt.Sprintf("builds/%d", -1), 200, &request, nil)
	shortToken := helpers.ShortenToken(runner.Token)

	switch result {
	case 404:
		// this is expected due to fact that we ask for non-existing job
		logrus.Println(shortToken, "Veryfing runner...", "is alive")
		return true
	case 403:
		logrus.Errorln(shortToken, "Veryfing runner...", "is removed")
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

	result, statusText := n.put(config.RunnerCredentials, fmt.Sprintf("builds/%d.json", id), 200, &request, nil)
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
