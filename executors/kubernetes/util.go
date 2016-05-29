package kubernetes

import (
	"fmt"
	"io"
	"time"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
)

// Stderr implements io.Reader and calls ErrFn on Write
type Stderr struct {
	ErrFn func(...interface{})
}

// Write calls ErrFn with p
func (s Stderr) Write(p []byte) (int, error) {
	s.ErrFn(string(p))
	return len(p), nil
}

func getKubeClientConfig(config *common.KubernetesConfig) (*restclient.Config, error) {
	switch {
	case len(config.CertFile) > 0:
		if len(config.KeyFile) == 0 || len(config.CAFile) == 0 {
			return nil, fmt.Errorf("ca file, cert file and key file must be specified when using file based auth")
		}
		return &restclient.Config{
			Host: config.Host,
			TLSClientConfig: restclient.TLSClientConfig{
				CertFile: config.CertFile,
				KeyFile:  config.KeyFile,
				CAFile:   config.CAFile,
			},
		}, nil
	case len(config.Host) > 0:
		return &restclient.Config{
			Host: config.Host,
		}, nil
	default:
		return restclient.InClusterConfig()
	}
}

func getKubeClient(config *common.KubernetesConfig) (*client.Client, error) {
	if kubeClient != nil {
		return kubeClient, nil
	}

	restConfig, err := getKubeClientConfig(config)
	if err != nil {
		return nil, err
	}

	return client.New(restConfig)
}

func waitForPodRunning(c *client.Client, pod *api.Pod, out io.Writer) (status api.PodPhase, err error) {
	for {
		pod, err := c.Pods(pod.Namespace).Get(pod.Name)
		if err != nil {
			return api.PodUnknown, err
		}
		ready := false
		if pod.Status.Phase == api.PodRunning {
			ready = true
			for _, status := range pod.Status.ContainerStatuses {
				if !status.Ready {
					ready = false
					break
				}
			}
			if ready {
				return api.PodRunning, nil
			}
		}
		if pod.Status.Phase == api.PodSucceeded || pod.Status.Phase == api.PodFailed {
			return pod.Status.Phase, nil
		}
		fmt.Fprintf(out, "Waiting for pod %s/%s to be running, status is %s, pod ready: %v\n", pod.Namespace, pod.Name, pod.Status.Phase, ready)
		time.Sleep(1 * time.Second)
		continue
	}
}
