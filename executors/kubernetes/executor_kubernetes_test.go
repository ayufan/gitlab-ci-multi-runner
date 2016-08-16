package kubernetes

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors"

	"github.com/Sirupsen/logrus"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
	"k8s.io/kubernetes/pkg/api/testapi"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/client/unversioned/fake"
)

var (
	TRUE  = true
	FALSE = false
)

func TestLimits(t *testing.T) {
	tests := []struct {
		CPU, Memory string
		Expected    api.ResourceList
		Error       bool
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
			Error:    true,
		},
		{
			Memory:   "100j",
			Expected: api.ResourceList{},
			Error:    true,
		},
		{
			Expected: api.ResourceList{},
		},
	}

	for _, test := range tests {
		res, err := limits(test.CPU, test.Memory)

		if err != nil && !test.Error {
			t.Errorf("got error but expected '%v': %s", test.Expected, err)
			continue
		}

		if !reflect.DeepEqual(res, test.Expected) {
			t.Errorf("got: '%v' but expected: '%v'", res, test.Expected)
			continue
		}
	}
}

func TestBuildContainer(t *testing.T) {
	tests := []struct {
		Name, Image, BuildDir string
		Privileged            bool
		Command               []string
		Environment           []string
		Limits                api.ResourceList

		Expected api.Container
	}{
		{
			Name:        "test",
			Image:       "image",
			BuildDir:    "/test/build",
			Privileged:  true,
			Command:     []string{"test", "command"},
			Environment: nil,
			Limits:      nil,

			Expected: api.Container{
				Name:    "test",
				Image:   "image",
				Command: []string{"test", "command"},
				Env: []api.EnvVar{
					{Name: "CI", Value: "true"}, {Name: "CI_BUILD_REF"}, {Name: "CI_BUILD_BEFORE_SHA"},
					{Name: "CI_BUILD_REF_NAME"}, {Name: "CI_BUILD_ID", Value: "0"}, {Name: "CI_BUILD_REPO"},
					{Name: "CI_BUILD_TOKEN"}, {Name: "CI_PROJECT_ID", Value: "0"}, {Name: "CI_PROJECT_DIR", Value: "/test/build"},
					{Name: "CI_SERVER", Value: "yes"}, {Name: "CI_SERVER_NAME", Value: "GitLab CI"}, {Name: "CI_SERVER_VERSION"},
					{Name: "CI_SERVER_REVISION"}, {Name: "GITLAB_CI", Value: "true"},
				},
				Resources: api.ResourceRequirements{
					Limits: nil,
				},
				VolumeMounts: []api.VolumeMount{
					api.VolumeMount{
						Name:      "repo",
						MountPath: "/test",
					},
					api.VolumeMount{
						Name:      "etc-ssl-certs",
						MountPath: "/etc/ssl/certs",
					},
					api.VolumeMount{
						Name:      "usr-share-ca-certificates",
						MountPath: "/usr/share/ca-certificates",
					},
				},
				SecurityContext: &api.SecurityContext{
					Privileged: &TRUE,
				},
				Stdin: true,
			},
		},
	}

	for _, test := range tests {
		e := executor{
			extraOptions: FakeOptions{
				privileged: test.Privileged,
			},
			AbstractExecutor: executors.AbstractExecutor{
				Build: &common.Build{
					BuildDir: test.BuildDir,
					Runner: &common.RunnerConfig{
						RunnerSettings: common.RunnerSettings{
							Environment: test.Environment,
						},
					},
				},
				ExecutorOptions: executors.ExecutorOptions{
					Shell: common.ShellScriptInfo{
						Build: &common.Build{
							BuildDir: test.BuildDir,
						},
					},
				},
			},
		}
		if bc := e.buildContainer(test.Name, test.Image, test.Limits, test.Command...); !reflect.DeepEqual(bc, test.Expected) {
			t.Errorf("error testing buildContainer. expected '%v', got '%v'", test.Expected, bc)
			continue
		}
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
						Host:            "test-server",
						ServiceCPUs:     "0.5",
						ServiceMemory:   "200Mi",
						CPUs:            "1.5",
						Memory:          "4Gi",
						AllowPrivileged: true,
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
				extraOptions: DefaultOptions{
					BuildVariables: []common.BuildVariable{
						common.BuildVariable{Key: "privileged", Value: "true"},
					},
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
						Host:            "test-server",
						ServiceCPUs:     "0.5",
						ServiceMemory:   "200Mi",
						CPUs:            "1.5",
						Memory:          "4Gi",
						AllowPrivileged: false,
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
		if !reflect.DeepEqual(e, test.Expected) {
			t.Errorf("Got executor '%+v' but expected '%+v'", e, test.Expected)
			continue
		}
	}
}

type FakeReadCloser struct {
	io.Reader
}

func (f FakeReadCloser) Close() error { return nil }

type FakeBuildTrace struct {
	testWriter
}

func (f FakeBuildTrace) Success()                  {}
func (f FakeBuildTrace) Fail(error)                {}
func (f FakeBuildTrace) Notify(func())             {}
func (f FakeBuildTrace) Aborted() chan interface{} { return make(chan interface{}) }
func (f FakeBuildTrace) IsStdout() bool            { return false }
