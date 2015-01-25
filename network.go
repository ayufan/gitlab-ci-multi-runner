package main

import (
	"encoding/json"
	"io/ioutil"
	"io"
	"errors"
	"net/http"
	"bytes"
	"fmt"

	log "github.com/Sirupsen/logrus"
)

type UpdateState int

const (
	UpdateSucceeded UpdateState = iota
	UpdateAbort
	UpdateFailed
)

type BuildState string

const (
	Pending BuildState	= "pending"
	Running				= "running"
	Failed				= "failed"
	Success				= "success"
)

type GetBuildRequest struct {
	Token                string     `json:"token,omitempty"`
}

type GetBuildResponse struct {
	Id                   int        `json:"token,omitempty"`
	ProjectId            int        `json:"project_id,omitempty"`
	Commands             string     `json:"commands,omitempty"`
	RepoURL              string     `json:"repo_url,omitempty"`
	Sha                  string     `json:"sha,omitempty"`
	RefName              string     `json:"ref,omitempty"`
	BeforeSha            string     `json:"before_sha,omitempty"`
	AllowGitFetch        bool       `json:"allow_git_fetch,omitempty"`
	Timeout              int        `json:"timeout,omitempty"`
}

type RegisterRunnerRequest struct {
	Token                string     `json:"token,omitempty"`
	Hostname             string     `json:"hostname,omitempty"`
}

type RegisterRunnerResponse struct {
	Token                string     `json:"token,omitempty"`
}

type UpdateBuildRequest struct {
	Token                string     `json:"token,omitempty"`
	State                BuildState `json:"state,omitempty"`
	Trace                string     `json:"trace,omitempty"`
}

func sendJsonRequest(url string, method string, statusCode int, request interface{}, response interface{}) int {
	var data *bytes.Reader

	if request != nil {
		body, err := json.Marshal(request)
		if err != nil {
			log.Errorf("Failed to marshal project object: %v", err)
			return -1
		}
		data = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, data)
	if err != nil {
		log.Errorf("Failed to create NewRequest", err)
		return -1
	}

	if request != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("Couldn't execute %v against %s: %v", req.Method, req.URL, err)
		return -1
	}
	defer res.Body.Close()

	if res.StatusCode == statusCode {
		if response != nil {
			d := json.NewDecoder(res.Body)
			err = d.Decode(response)
			if err != nil {
				log.Errorf("Error decoding json payload %v", err)
				return -1
			}
		}
	}

	return res.StatusCode
}

func getJson(url string, statusCode int, response interface{}) int {
	return sendJsonRequest(url, "GET", statusCode, nil, response)
}

func postJson(url string, statusCode int, request interface{}, response interface{}) int {
	return sendJsonRequest(url, "POST", statusCode, request, response)
}

func putJson(url string, statusCode int, request interface{}, response interface{}) int {
	return sendJsonRequest(url, "PUT", statusCode, request, response)
}

func readPayload(r io.Reader) ([]byte, error) {
	maxPayloadSize := int64(1 << 63 - 1)
	maxPayloadSize = int64(10 << 20) // 10 MB is a lot of text.
	b, err := ioutil.ReadAll(io.LimitReader(r, maxPayloadSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > maxPayloadSize {
		err = errors.New("http: POST too large")
		return nil, err
	}
	return b, nil
}

func getUrl(baseURL string, request string, a ...interface{}) string {
	return fmt.Sprintf("%s/api/v1/%s", baseURL, fmt.Sprintf(request, a...));
}

func GetBuild(config *RunnerConfig) *GetBuildResponse {
	request := GetBuildRequest{
		Token: config.Token,
	}

	var response GetBuildResponse
	result := postJson(getUrl(config.URL, "builds/register.json"), 201, &request, &response)

	switch result {
	case 201:	return &response
	case 403:	return nil
	default:	return nil
	}
}

func RegisterRunner(config *RunnerConfig) *RegisterRunnerResponse {
	request := RegisterRunnerRequest{
		Token: config.Token,
		Hostname: config.Name,
	}

	var response RegisterRunnerResponse
	result := postJson(getUrl(config.URL, "runners/register.json"), 201, &request, &response)

	switch result {
	case 201:	return &response
	default:	return nil
	}
}

func UpdateBuild(config *RunnerConfig, id int, state BuildState, trace io.Reader) UpdateState {
	data, err := readPayload(trace)
	if err != nil {
		return UpdateFailed
	}

	request := UpdateBuildRequest{
		Token: config.Token,
		State: state,
		Trace: string(data),
	}

	result := putJson(getUrl(config.URL, "builds/%d.json", id), 201, &request, nil)
	switch result {
	case 201: return UpdateSucceeded
	case 404: return UpdateAbort
	default:  return UpdateFailed
	}
}
