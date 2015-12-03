package network

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	. "gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

var brokenCredentials RunnerCredentials = RunnerCredentials{
	URL: "broken",
}

var brokenConfig RunnerConfig = RunnerConfig{
	RunnerCredentials: brokenCredentials,
}

func TestClients(t *testing.T) {
	c := GitLabClient{}
	c1, _ := c.getClient(RunnerCredentials{
		URL: "http://test/",
	})
	c2, _ := c.getClient(RunnerCredentials{
		URL: "http://test2/",
	})
	c4, _ := c.getClient(RunnerCredentials{
		URL:       "http://test/",
		TLSCAFile: "ca_file",
	})
	c5, _ := c.getClient(RunnerCredentials{
		URL:       "http://test/",
		TLSCAFile: "ca_file",
	})
	c6, c6err := c.getClient(brokenCredentials)
	assert.NotEqual(t, c1, c2)
	assert.NotEqual(t, c1, c4)
	assert.Equal(t, c4, c5)
	assert.Nil(t, c6)
	assert.Error(t, c6err)
}

func TestGetBuild(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/builds/register.json" {
			w.WriteHeader(404)
			return
		}

		if r.Method != "POST" {
			w.WriteHeader(406)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)

		var req map[string]interface{}
		err = json.Unmarshal(body, &req)
		assert.NoError(t, err)

		res := make(map[string]interface{})

		switch req["token"].(string) {
		case "valid":
			res["id"] = 10
		case "no-builds":
			w.WriteHeader(404)
			return
		case "invalid":
			w.WriteHeader(403)
			return
		default:
			w.WriteHeader(400)
			return
		}

		if r.Header.Get("Accept") != "application/json" {
			w.WriteHeader(400)
			return
		}

		output, err := json.Marshal(res)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		w.Write(output)
	}

	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	validToken := RunnerConfig{
		RunnerCredentials: RunnerCredentials{
			URL:   s.URL,
			Token: "valid",
		},
	}

	noBuildsToken := RunnerConfig{
		RunnerCredentials: RunnerCredentials{
			URL:   s.URL,
			Token: "no-builds",
		},
	}

	invalidToken := RunnerConfig{
		RunnerCredentials: RunnerCredentials{
			URL:   s.URL,
			Token: "invalid",
		},
	}

	c := GitLabClient{}

	res, ok := c.GetBuild(validToken)
	if assert.NotNil(t, res) {
		assert.NotEmpty(t, res.ID)
	}
	assert.True(t, ok)

	res, ok = c.GetBuild(noBuildsToken)
	assert.Nil(t, res)
	assert.True(t, ok, "If no builds, runner is healthy")

	res, ok = c.GetBuild(invalidToken)
	assert.Nil(t, res)
	assert.False(t, ok, "If token is invalid, the runner is unhealthy")

	res, ok = c.GetBuild(brokenConfig)
	assert.Nil(t, res)
	assert.False(t, ok)
}

func TestRegisterRunner(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/runners/register.json" {
			w.WriteHeader(404)
			return
		}

		if r.Method != "POST" {
			w.WriteHeader(406)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)

		var req map[string]interface{}
		err = json.Unmarshal(body, &req)
		assert.NoError(t, err)

		res := make(map[string]interface{})

		switch req["token"].(string) {
		case "valid":
			if req["description"].(string) != "test" {
				w.WriteHeader(400)
				return
			}

			res["token"] = req["token"].(string)
		case "invalid":
			w.WriteHeader(403)
			return
		default:
			w.WriteHeader(400)
			return
		}

		if r.Header.Get("Accept") != "application/json" {
			w.WriteHeader(400)
			return
		}

		output, err := json.Marshal(res)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		w.Write(output)
	}

	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	validToken := RunnerCredentials{
		URL:   s.URL,
		Token: "valid",
	}

	invalidToken := RunnerCredentials{
		URL:   s.URL,
		Token: "invalid",
	}

	otherToken := RunnerCredentials{
		URL:   s.URL,
		Token: "other",
	}

	c := GitLabClient{}

	res := c.RegisterRunner(validToken, "test", "tags")
	if assert.NotNil(t, res) {
		assert.Equal(t, validToken.Token, res.Token)
	}

	res = c.RegisterRunner(validToken, "invalid description", "tags")
	assert.Nil(t, res)

	res = c.RegisterRunner(invalidToken, "test", "tags")
	assert.Nil(t, res)

	res = c.RegisterRunner(otherToken, "test", "tags")
	assert.Nil(t, res)

	res = c.RegisterRunner(brokenCredentials, "test", "tags")
	assert.Nil(t, res)
}

func TestDeleteRunner(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/runners/delete" {
			w.WriteHeader(404)
			return
		}

		if r.Method != "DELETE" {
			w.WriteHeader(406)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)

		var req map[string]interface{}
		err = json.Unmarshal(body, &req)
		assert.NoError(t, err)

		switch req["token"].(string) {
		case "valid":
			w.WriteHeader(200)
		case "invalid":
			w.WriteHeader(403)
		default:
			w.WriteHeader(400)
		}
	}

	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	validToken := RunnerCredentials{
		URL:   s.URL,
		Token: "valid",
	}

	invalidToken := RunnerCredentials{
		URL:   s.URL,
		Token: "invalid",
	}

	otherToken := RunnerCredentials{
		URL:   s.URL,
		Token: "other",
	}

	c := GitLabClient{}

	state := c.DeleteRunner(validToken)
	assert.True(t, state)

	state = c.DeleteRunner(invalidToken)
	assert.False(t, state)

	state = c.DeleteRunner(otherToken)
	assert.False(t, state)

	state = c.DeleteRunner(brokenCredentials)
	assert.False(t, state)
}

func TestVerifyRunner(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/builds/-1" {
			w.WriteHeader(404)
			return
		}

		if r.Method != "PUT" {
			w.WriteHeader(406)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)

		var req map[string]interface{}
		err = json.Unmarshal(body, &req)
		assert.NoError(t, err)

		switch req["token"].(string) {
		case "valid":
			w.WriteHeader(404) // since the build id is broken, we should not find this build
		case "invalid":
			w.WriteHeader(403)
		default:
			w.WriteHeader(400)
		}
	}

	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	validToken := RunnerCredentials{
		URL:   s.URL,
		Token: "valid",
	}

	invalidToken := RunnerCredentials{
		URL:   s.URL,
		Token: "invalid",
	}

	otherToken := RunnerCredentials{
		URL:   s.URL,
		Token: "other",
	}

	c := GitLabClient{}

	state := c.VerifyRunner(validToken)
	assert.True(t, state)

	state = c.VerifyRunner(invalidToken)
	assert.False(t, state)

	state = c.VerifyRunner(otherToken)
	assert.True(t, state, "in other cases where we can't explicitly say that runner is valid we say that it's")

	state = c.VerifyRunner(brokenCredentials)
	assert.False(t, state)
}

func TestUpdateBuild(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/builds/10.json" {
			w.WriteHeader(404)
			return
		}

		if r.Method != "PUT" {
			w.WriteHeader(406)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)

		var req map[string]interface{}
		err = json.Unmarshal(body, &req)
		assert.NoError(t, err)

		assert.Equal(t, "token", req["token"])
		assert.Equal(t, "trace", req["trace"])

		switch req["state"].(string) {
		case "running":
			w.WriteHeader(200)
		case "forbidden":
			w.WriteHeader(403)
		default:
			w.WriteHeader(400)
		}
	}

	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	config := RunnerConfig{
		RunnerCredentials: RunnerCredentials{
			URL:   s.URL,
			Token: "token",
		},
	}

	c := GitLabClient{}

	state := c.UpdateBuild(config, 10, "running", "trace")
	assert.Equal(t, UpdateSucceeded, state, "Update should continue when running")

	state = c.UpdateBuild(config, 10, "forbidden", "trace")
	assert.Equal(t, UpdateAbort, state, "Update should if the state is forbidden")

	state = c.UpdateBuild(config, 10, "other", "trace")
	assert.Equal(t, UpdateFailed, state, "Update should fail for badly formatted request")

	state = c.UpdateBuild(config, 4, "state", "trace")
	assert.Equal(t, UpdateAbort, state, "Update should abort for unknown build")

	state = c.UpdateBuild(brokenConfig, 4, "state", "trace")
	assert.Equal(t, UpdateAbort, state)
}

func TestArtifactsUrl(t *testing.T) {
	c := GitLabClient{}
	url := c.GetArtifactsUploadURL(RunnerCredentials{
		URL: "http://test/",
	}, 10)
	assert.Equal(t, "http://test/api/v1/builds/10/artifacts", url)

	url = c.GetArtifactsUploadURL(brokenCredentials, 10)
	assert.Empty(t, url)
}
