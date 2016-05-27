package kubernetes

import (
	"fmt"
	"io"
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
)

func getKubeClientConfig(proxyURL string) (*restclient.Config, error) {
	if proxyURL != "" {
		return &restclient.Config{
			Host: proxyURL,
		}, nil
	}
	return restclient.InClusterConfig()
}

func getKubeClient(proxyURL string) (*client.Client, error) {
	config, err := getKubeClientConfig(proxyURL)
	if err != nil {
		return nil, err
	}

	return client.New(config)
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
		time.Sleep(2 * time.Second)
		continue
	}
}
