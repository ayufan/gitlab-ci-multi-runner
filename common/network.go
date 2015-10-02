package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"runtime"
)

type UpdateState int

const (
	UpdateSucceeded UpdateState = iota
	UpdateAbort
	UpdateFailed
)

type FeaturesInfo struct {
	Variables bool `json:"variables"`
	Image     bool `json:"image"`
	Services  bool `json:"services"`
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

type BuildVariable struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Public bool   `json:"public"`
}

type BuildOptions map[string]interface{}

type GetBuildResponse struct {
	ID            int             `json:"id,omitempty"`
	ProjectID     int             `json:"project_id,omitempty"`
	Commands      string          `json:"commands,omitempty"`
	RepoURL       string          `json:"repo_url,omitempty"`
	Sha           string          `json:"sha,omitempty"`
	RefName       string          `json:"ref,omitempty"`
	BeforeSha     string          `json:"before_sha,omitempty"`
	AllowGitFetch bool            `json:"allow_git_fetch,omitempty"`
	Timeout       int             `json:"timeout,omitempty"`
	Variables     []BuildVariable `json:"variables"`
	Options       BuildOptions    `json:"options"`
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

type UpdateBuildRequest struct {
	Info  VersionInfo `json:"info,omitempty"`
	Token string      `json:"token,omitempty"`
	State BuildState  `json:"state,omitempty"`
	Trace string      `json:"trace,omitempty"`
}

func sendJSONRequest(url string, method string, statusCode int, request interface{}, response interface{}) (int, string) {
	var body []byte
	var err error

	if request != nil {
		body, err = json.Marshal(request)
		if err != nil {
			return -1, fmt.Sprintf("failed to marshal project object: %v", err)
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return -1, fmt.Sprintf("failed to create NewRequest: %v", err)
	}

	if request != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return -1, fmt.Sprintf("couldn't execute %v against %s: %v", req.Method, req.URL, err)
	}
	defer res.Body.Close()

	if res.StatusCode == statusCode {
		if response != nil {
			d := json.NewDecoder(res.Body)
			err = d.Decode(response)
			if err != nil {
				return -1, fmt.Sprintf("Error decoding json payload %v", err)
			}
		}
	}

	return res.StatusCode, res.Status
}

func getJSON(url string, statusCode int, response interface{}) (int, string) {
	return sendJSONRequest(url, "GET", statusCode, nil, response)
}

func postJSON(url string, statusCode int, request interface{}, response interface{}) (int, string) {
	return sendJSONRequest(url, "POST", statusCode, request, response)
}

func putJSON(url string, statusCode int, request interface{}, response interface{}) (int, string) {
	return sendJSONRequest(url, "PUT", statusCode, request, response)
}

func deleteJSON(url string, statusCode int, response interface{}) (int, string) {
	return sendJSONRequest(url, "DELETE", statusCode, nil, response)
}

func getURL(baseURL string, request string, a ...interface{}) string {
	return fmt.Sprintf("%s/api/v1/%s", baseURL, fmt.Sprintf(request, a...))
}

func GetRunnerVersion(executor string) VersionInfo {
	info := VersionInfo{
		Name:         NAME,
		Version:      VERSION,
		Revision:     REVISION,
		Platform:     runtime.GOOS,
		Architecture: runtime.GOARCH,
		Executor:     executor,
		Features: FeaturesInfo{
			Variables: true,
			Image:     true,
			Services:  true,
		},
	}

	if features := GetExecutorFeatures(executor); features != nil {
		info.Features = *features
	}

	return info
}

func GetBuild(config RunnerConfig) (*GetBuildResponse, bool) {
	request := GetBuildRequest{
		Info:  GetRunnerVersion(config.Executor),
		Token: config.Token,
	}

	var response GetBuildResponse
	result, statusText := postJSON(getURL(config.URL, "builds/register.json"), 201, &request, &response)

	switch result {
	case 201:
		log.Println(config.ShortDescription(), "Checking for builds...", "received")
		return &response, true
	case 403:
		log.Errorln(config.ShortDescription(), "Checking for builds...", "forbidden")
		return nil, false
	case 404:
		log.Debugln(config.ShortDescription(), "Checking for builds...", "nothing")
		return nil, true
	default:
		log.Warningln(config.ShortDescription(), "Checking for builds...", "failed:", statusText)
		return nil, true
	}
}

func RegisterRunner(url, token, description, tags string) *RegisterRunnerResponse {
	// TODO: pass executor
	request := RegisterRunnerRequest{
		Info:        GetRunnerVersion(""),
		Token:       token,
		Description: description,
		Tags:        tags,
	}

	var response RegisterRunnerResponse
	result, statusText := postJSON(getURL(url, "runners/register.json"), 201, &request, &response)
	shortToken := helpers.ShortenToken(token)

	switch result {
	case 201:
		log.Println(shortToken, "Registering runner...", "succeeded")
		return &response
	case 403:
		log.Errorln(shortToken, "Registering runner...", "forbidden (check registration token)")
		return nil
	default:
		log.Errorln(shortToken, "Registering runner...", "failed", statusText)
		return nil
	}
}

func DeleteRunner(url, token string) bool {
	result, statusText := deleteJSON(getURL(url, "runners/delete?token=%v", token), 200, nil)
	shortToken := helpers.ShortenToken(token)

	switch result {
	case 200:
		log.Println(shortToken, "Deleting runner...", "succeeded")
		return true
	case 403:
		log.Errorln(shortToken, "Deleting runner...", "forbidden")
		return false
	default:
		log.Errorln(shortToken, "Deleting runner...", "failed", statusText)
		return false
	}
}

func VerifyRunner(url, token string) bool {
	result, statusText := putJSON(getURL(url, "builds/%v?token=%v", -1, token), 200, nil, nil)
	shortToken := helpers.ShortenToken(token)

	switch result {
	case 404:
		// this is expected due to fact that we ask for non-existing job
		log.Println(shortToken, "Veryfing runner...", "is alive")
		return true
	case 403:
		log.Errorln(shortToken, "Veryfing runner...", "is removed")
		return false
	default:
		log.Errorln(shortToken, "Veryfing runner...", "failed", statusText)
		return true
	}
}

func UpdateBuild(config RunnerConfig, id int, state BuildState, trace string) UpdateState {
	request := UpdateBuildRequest{
		Info:  GetRunnerVersion(config.Executor),
		Token: config.Token,
		State: state,
		Trace: trace,
	}

	result, statusText := putJSON(getURL(config.URL, "builds/%d.json", id), 200, &request, nil)
	switch result {
	case 200:
		log.Println(config.ShortDescription(), id, "Submitting build to coordinator...", "ok")
		return UpdateSucceeded
	case 404:
		log.Warningln(config.ShortDescription(), id, "Submitting build to coordinator...", "aborted")
		return UpdateAbort
	case 403:
		log.Errorln(config.ShortDescription(), id, "Submitting build to coordinator...", "forbidden")
		return UpdateAbort
	default:
		log.Warningln(config.ShortDescription(), id, "Submitting build to coordinator...", "failed", statusText)
		return UpdateFailed
	}
}
