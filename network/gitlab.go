package network

import (
	"fmt"
	. "gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
<<<<<<< HEAD
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
=======
>>>>>>> upstream/master
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
	key := fmt.Sprintf("%s_%s", runner.URL, runner.TLSCAFile)
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

func (n *GitLabClient) do(runner RunnerCredentials, method, uri string, statusCode int, request interface{}, response interface{}) (int, string, string) {
	c, err := n.getClient(runner)
	if err != nil {
		return clientError, err.Error(), ""
	}

	return c.do(uri, prepRequest, statusCode, response)
}

func (n *GitLabClient) doJson(runner RunnerCredentials, method, uri string, statusCode int, request interface{}, response interface{}) (int, string, string) {
	c, err := n.getClient(runner)
	if err != nil {
		return clientError, err.Error(), ""
	}

	return c.doJson(uri, method, statusCode, request, response)
}

func (n *GitLabClient) GetBuild(config RunnerConfig) (*GetBuildResponse, bool) {
	request := GetBuildRequest{
		Info:  n.getRunnerVersion(config),
		Token: config.Token,
	}

	var response GetBuildResponse
	result, statusText, _ := n.doJson(config.RunnerCredentials, "POST", "builds/register.json", 201, &request, &response)

	switch result {
	case 201:
		config.Log().Println("Checking for builds...", "received")
		response.TLSCAChain = certificates
		return &response, true
	case 403:
		config.Log().Errorln("Checking for builds...", "forbidden")
		return nil, false
	case 404:
		config.Log().Debugln("Checking for builds...", "nothing")
		return nil, true
	case clientError:
		config.Log().WithField("status", statusText).Errorln("Checking for builds...", "error")
		return nil, false
	default:
		config.Log().WithField("status", statusText).Warningln("Checking for builds...", "failed")
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
	result, statusText, _ := n.doJson(runner, "POST", "runners/register.json", 201, &request, &response)

	switch result {
	case 201:
		runner.Log().Println("Registering runner...", "succeeded")
		return &response
	case 403:
		runner.Log().Errorln("Registering runner...", "forbidden (check registration token)")
		return nil
	case clientError:
		runner.Log().WithField("status", statusText).Errorln("Registering runner...", "error")
		return nil
	default:
		runner.Log().WithField("status", statusText).Errorln("Registering runner...", "failed")
		return nil
	}
}

func (n *GitLabClient) DeleteRunner(runner RunnerCredentials) bool {
	request := DeleteRunnerRequest{
		Token: runner.Token,
	}

	result, statusText, _ := n.doJson(runner, "DELETE", "runners/delete", 200, &request, nil)

	switch result {
	case 200:
		runner.Log().Println("Deleting runner...", "succeeded")
		return true
	case 403:
		runner.Log().Errorln("Deleting runner...", "forbidden")
		return false
	case clientError:
		runner.Log().WithField("status", statusText).Errorln("Deleting runner...", "error")
		return false
	default:
		runner.Log().WithField("status", statusText).Errorln("Deleting runner...", "failed")
		return false
	}
}

func (n *GitLabClient) VerifyRunner(runner RunnerCredentials) bool {
	request := VerifyRunnerRequest{
		Token: runner.Token,
	}

	// HACK: we use non-existing build id to check if receive forbidden or not found
	result, statusText, _ := n.doJson(runner, "PUT", fmt.Sprintf("builds/%d", -1), 200, &request, nil)

	switch result {
	case 404:
		// this is expected due to fact that we ask for non-existing job
		runner.Log().Println("Veryfing runner...", "is alive")
		return true
	case 403:
		runner.Log().Errorln("Veryfing runner...", "is removed")
		return false
	case clientError:
		runner.Log().WithField("status", statusText).Errorln("Veryfing runner...", "error")
		return false
	default:
		runner.Log().WithField("status", statusText).Errorln("Veryfing runner...", "failed")
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

	result, statusText, _ := n.doJson(config.RunnerCredentials, "PUT", fmt.Sprintf("builds/%d.json", id), 200, &request, nil)
	switch result {
	case 200:
		config.Log().Println(id, "Submitting build to coordinator...", "ok")
		return UpdateSucceeded
	case 404:
		config.Log().Warningln(id, "Submitting build to coordinator...", "aborted")
		return UpdateAbort
	case 403:
		config.Log().Errorln(id, "Submitting build to coordinator...", "forbidden")
		return UpdateAbort
	case clientError:
		config.Log().WithField("status", statusText).Errorln(id, "Submitting build to coordinator...", "error")
		return UpdateAbort
	default:
		config.Log().WithField("status", statusText).Warningln(id, "Submitting build to coordinator...", "failed")
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
	// Create this here so that we can dispose of it after the request has been sent
	tempFile, err := ioutil.TempFile("", "artifacts_")
	if err != nil {
		logrus.Warningln(config.ShortDescription(), id, "Uploading artifacts to coordinator...", "failed", "failed to create temp upload file")
	}
	
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())
	
	result, statusText := n.do(config.RunnerCredentials, n.GetArtifactsUploadURL(config.RunnerCredentials, id), func(url string) (*http.Request, error) {
		mpw := multipart.NewWriter(tempFile)
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

		if _, err := tempFile.Seek(0, 0); err != nil {
			return nil, err
		}
		
		fStat, err := os.Stat(tempFile.Name())
		if err != nil {
			return nil, err
		}
		
		req, err := http.NewRequest("POST", url, tempFile)
		if err != nil {
			return nil, err
		}
		
		req.Header.Set("Content-Length", string(fStat.Size()))
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
