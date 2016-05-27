package kubernetes

import (
	"fmt"
	"io"
	"net/url"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/client/unversioned/remotecommand"
)

// RemoteAttach defines the interface accepted by the Attach command - provided for test stubbing
type RemoteAttach interface {
	Attach(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error
}

// DefaultRemoteAttach is the standard implementation of attaching
type DefaultRemoteAttach struct{}

func (*DefaultRemoteAttach) Attach(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	exec, err := remotecommand.NewExecutor(config, method, url)
	if err != nil {
		return err
	}
	return exec.Stream(stdin, stdout, stderr, tty)
}

type AttachOptions struct {
	Pod           *api.Pod
	ContainerName string

	In  io.Reader
	Out io.Writer
	Err io.Writer
	TTY bool

	Client *client.Client
	Config *restclient.Config
	Attach RemoteAttach
}

// GetContainer returns the container to attach to, with a fallback.
func (a *AttachOptions) GetContainer(pod *api.Pod) api.Container {
	if len(a.ContainerName) > 0 {
		for _, container := range pod.Spec.Containers {
			if container.Name == a.ContainerName {
				return container
			}
		}
	}

	return a.Pod.Spec.Containers[0]
}

func (a *AttachOptions) Run() error {
	if a.Pod == nil {
		return fmt.Errorf("pod must not be nil")
	}
	// check for TTY
	tty := a.TTY

	containerToAttach := a.GetContainer(a.Pod)

	if tty && !containerToAttach.TTY {
		tty = false
	}
	if a.In != nil {
		t.In = a.In
		if tty && !t.IsTerminal() {
			tty = false
		}
	}
	t.Raw = tty

	fn := func() error {
		req := a.Client.RESTClient.Post().
			Resource("pods").
			Name(a.Pod.Name).
			Namespace(a.Pod.Namespace).
			SubResource("attach")
		req.VersionedParams(&api.PodAttachOptions{
			Container: containerToAttach.Name,
			Stdin:     a.In != nil,
			Stdout:    a.Out != nil,
			Stderr:    a.Err != nil,
			TTY:       tty,
		}, api.ParameterCodec)

		return a.Attach.Attach("POST", req.URL(), a.Config, a.In, a.Out, a.Err, tty)
	}

	if err := t.Safe(fn); err != nil {
		return err
	}

	return nil
}
