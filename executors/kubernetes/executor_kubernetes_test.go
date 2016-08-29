package kubernetes

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
	"k8s.io/kubernetes/pkg/api/testapi"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/client/unversioned/fake"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
)

var (
	TRUE = true
)

func TestLimits(t *testing.T) {
	tests := []struct {
		CPU, Memory string
		Expected    api.ResourceList
	}{
		{
			CPU:    "100m",
			Memory: "100Mi",
			Expected: api.ResourceList{
				api.ResourceLimitsCPU:    resource.MustParse("100m"),
				api.ResourceLimitsMemory: resource.MustParse("100Mi"),
			},
		},
		{
			CPU: "100m",
			Expected: api.ResourceList{
				api.ResourceLimitsCPU: resource.MustParse("100m"),
			},
		},
		{
			Memory: "100Mi",
			Expected: api.ResourceList{
				api.ResourceLimitsMemory: resource.MustParse("100Mi"),
			},
		},
		{
			CPU:      "100j",
			Expected: api.ResourceList{},
		},
		{
			Memory:   "100j",
			Expected: api.ResourceList{},
		},
		{
			Expected: api.ResourceList{},
		},
	}

	for _, test := range tests {
		res, _ := limits(test.CPU, test.Memory)
		assert.Equal(t, test.Expected, res)
	}
}

func TestCleanup(t *testing.T) {
	version := testapi.Default.GroupVersion().Version
	codec := testapi.Default.Codec()

	tests := []struct {
		Pod        *api.Pod
		ClientFunc func(*http.Request) (*http.Response, error)
		Error      bool
	}{
		{
			Pod: &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-ns",
				},
			},
			ClientFunc: func(req *http.Request) (*http.Response, error) {
				switch p, m := req.URL.Path, req.Method; {
				case m == "DELETE" && p == "/api/"+version+"/namespaces/test-ns/pods/test-pod":
					return &http.Response{StatusCode: 200, Body: FakeReadCloser{
						Reader: strings.NewReader(""),
					}}, nil
				default:
					return nil, fmt.Errorf("unexpected request. method: %s, path: %s", m, p)
				}
			},
		},
		{
			Pod: &api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-ns",
				},
			},
			ClientFunc: func(req *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("delete failed")
			},
			Error: true,
		},
	}

	for _, test := range tests {
		c := client.NewOrDie(&restclient.Config{ContentConfig: restclient.ContentConfig{GroupVersion: &unversioned.GroupVersion{Version: version}}})
		fakeClient := fake.RESTClient{
			Codec:  codec,
			Client: fake.CreateHTTPClient(test.ClientFunc),
		}
		c.Client = fakeClient.Client

		ex := executor{
			kubeClient: c,
			pod:        test.Pod,
		}
		errored := false
		buildTrace := FakeBuildTrace{
			testWriter{
				call: func(b []byte) (int, error) {
					if test.Error && !errored {
						if s := string(b); strings.Contains(s, "Error cleaning up") {
							errored = true
						} else {
							t.Errorf("expected failure. got: '%s'", string(b))
						}
					}
					return len(b), nil
				},
			},
		}
		ex.AbstractExecutor.BuildTrace = buildTrace
		ex.AbstractExecutor.BuildLogger = common.NewBuildLogger(buildTrace, logrus.WithFields(logrus.Fields{}))
		ex.Cleanup()
		if test.Error && !errored {
			t.Errorf("expected cleanup to error but it didn't")
		}
	}
}

func TestPrepare(t *testing.T) {
	tests := []struct {
		GlobalConfig *common.Config
		RunnerConfig *common.RunnerConfig
		Build        *common.Build

		Expected *executor
		Error    bool
	}{
		{
			GlobalConfig: &common.Config{},
			RunnerConfig: &common.RunnerConfig{
				RunnerSettings: common.RunnerSettings{
					Kubernetes: &common.KubernetesConfig{
						Host:          "test-server",
						ServiceCPUs:   "0.5",
						ServiceMemory: "200Mi",
						CPUs:          "1.5",
						Memory:        "4Gi",
						Privileged:    true,
					},
				},
			},
			Build: &common.Build{
				GetBuildResponse: common.GetBuildResponse{
					Sha: "1234567890",
					Options: common.BuildOptions{
						"image": "test-image",
					},
					Variables: []common.BuildVariable{
						common.BuildVariable{Key: "privileged", Value: "true"},
					},
				},
				Runner: &common.RunnerConfig{},
			},
			Expected: &executor{
				options: &kubernetesOptions{
					Image: "test-image",
				},
				serviceLimits: api.ResourceList{
					api.ResourceLimitsCPU:    resource.MustParse("0.5"),
					api.ResourceLimitsMemory: resource.MustParse("200Mi"),
				},
				buildLimits: api.ResourceList{
					api.ResourceLimitsCPU:    resource.MustParse("1.5"),
					api.ResourceLimitsMemory: resource.MustParse("4Gi"),
				},
			},
		},
		{
			GlobalConfig: &common.Config{},
			RunnerConfig: &common.RunnerConfig{
				RunnerSettings: common.RunnerSettings{
					Kubernetes: &common.KubernetesConfig{
						Host:          "test-server",
						ServiceCPUs:   "0.5",
						ServiceMemory: "200Mi",
						CPUs:          "1.5",
						Memory:        "4Gi",
						Privileged:    false,
					},
				},
			},
			Build: &common.Build{
				GetBuildResponse: common.GetBuildResponse{
					Sha: "1234567890",
					Options: common.BuildOptions{
						"image": "test-image",
					},
				},
				Runner: &common.RunnerConfig{},
			},
			Expected: &executor{
				options: &kubernetesOptions{
					Image: "test-image",
				},
				serviceLimits: api.ResourceList{
					api.ResourceLimitsCPU:    resource.MustParse("0.5"),
					api.ResourceLimitsMemory: resource.MustParse("200Mi"),
				},
				buildLimits: api.ResourceList{
					api.ResourceLimitsCPU:    resource.MustParse("1.5"),
					api.ResourceLimitsMemory: resource.MustParse("4Gi"),
				},
			},
			Error: true,
		},
	}

	for _, test := range tests {
		e := &executor{
			AbstractExecutor: executors.AbstractExecutor{
				ExecutorOptions: executorOptions,
			},
		}

		err := e.Prepare(test.GlobalConfig, test.RunnerConfig, test.Build)

		if err != nil {
			if test.Error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if !test.Error {
				t.Errorf("Got error. Expected: %v", test.Expected)
			}
			continue
		}

		// Set this to nil so we aren't testing the functionality of the
		// base AbstractExecutor's Prepare method
		e.AbstractExecutor = executors.AbstractExecutor{}

		// TODO: Improve this so we don't have to nil-ify the kubeClient.
		// It currently contains some moving parts that are failing, meaning
		// we'll need to mock _something_
		e.kubeClient = nil
		assert.Equal(t, test.Expected, e)
	}
}

func TestKubernetesSuccessRun(t *testing.T) {
	if helpers.SkipIntegrationTests(t, "kubectl", "cluster-info") {
		return
	}

	buildResponse := common.SuccessfulBuild
	buildResponse.Options = map[string]interface{}{
		"image": "docker:git",
	}
	build := &common.Build{
		GetBuildResponse: buildResponse,
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor:   "kubernetes",
				Kubernetes: &common.KubernetesConfig{},
			},
		},
	}

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	assert.NoError(t, err)
}

func TestKubernetesBuildFail(t *testing.T) {
	if helpers.SkipIntegrationTests(t, "kubectl", "cluster-info") {
		return
	}

	build := &common.Build{
		GetBuildResponse: common.FailedBuild,
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor:   "kubernetes",
				Kubernetes: &common.KubernetesConfig{},
			},
		},
	}
	build.Options = map[string]interface{}{
		"image": "docker:git",
	}

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	require.Error(t, err, "error")
	assert.IsType(t, err, &common.BuildError{})
	assert.Contains(t, err.Error(), "Error executing in Docker Container: 1")
}

func TestKubernetesMissingImage(t *testing.T) {
	if helpers.SkipIntegrationTests(t, "kubectl", "cluster-info") {
		return
	}

	build := &common.Build{
		GetBuildResponse: common.FailedBuild,
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor:   "kubernetes",
				Kubernetes: &common.KubernetesConfig{},
			},
		},
	}
	build.Options = map[string]interface{}{
		"image": "some/non-existing/image",
	}

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestKubernetesBuildAbort(t *testing.T) {
	if helpers.SkipIntegrationTests(t, "kubectl", "cluster-info") {
		return
	}

	build := &common.Build{
		GetBuildResponse: common.FailedBuild,
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor:   "kubernetes",
				Kubernetes: &common.KubernetesConfig{},
			},
		},
		SystemInterrupt: make(chan os.Signal, 1),
	}
	build.Options = map[string]interface{}{
		"image": "docker:git",
	}

	abortTimer := time.AfterFunc(time.Second, func() {
		t.Log("Interrupt")
		build.SystemInterrupt <- os.Interrupt
	})
	defer abortTimer.Stop()

	timeoutTimer := time.AfterFunc(time.Minute, func() {
		t.Log("Timedout")
		t.FailNow()
	})
	defer timeoutTimer.Stop()

	err := build.Run(&common.Config{}, &common.Trace{Writer: os.Stdout})
	assert.EqualError(t, err, "aborted: interrupt")
}

func TestKubernetesBuildCancel(t *testing.T) {
	if helpers.SkipIntegrationTests(t, "kubectl", "cluster-info") {
		return
	}

	build := &common.Build{
		GetBuildResponse: common.FailedBuild,
		Runner: &common.RunnerConfig{
			RunnerSettings: common.RunnerSettings{
				Executor:   "kubernetes",
				Kubernetes: &common.KubernetesConfig{},
			},
		},
		SystemInterrupt: make(chan os.Signal, 1),
	}
	build.Options = map[string]interface{}{
		"image": "docker:git",
	}

	trace := &common.Trace{Writer: os.Stdout, Abort: make(chan interface{}, 1)}

	abortTimer := time.AfterFunc(time.Second, func() {
		t.Log("Interrupt")
		trace.Abort <- true
	})
	defer abortTimer.Stop()

	timeoutTimer := time.AfterFunc(time.Minute, func() {
		t.Log("Timedout")
		t.FailNow()
	})
	defer timeoutTimer.Stop()

	err := build.Run(&common.Config{}, trace)
	assert.IsType(t, err, &common.BuildError{})
	assert.EqualError(t, err, "canceled")
}

type FakeReadCloser struct {
	io.Reader
}

func (f FakeReadCloser) Close() error {
	return nil
}

type FakeBuildTrace struct {
	testWriter
}

func (f FakeBuildTrace) Success()      {}
func (f FakeBuildTrace) Fail(error)    {}
func (f FakeBuildTrace) Notify(func()) {}
func (f FakeBuildTrace) Aborted() chan interface{} {
	return make(chan interface{})
}
func (f FakeBuildTrace) IsStdout() bool {
	return false
}
