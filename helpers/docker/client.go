package docker_helpers

import "github.com/fsouza/go-dockerclient"

type Client interface {
	InspectImage(name string) (*docker.Image, error)
	PullImage(opts docker.PullImageOptions, auth docker.AuthConfiguration) error
	ImportImage(opts docker.ImportImageOptions) error

	CreateContainer(opts docker.CreateContainerOptions) (*docker.Container, error)
	StartContainer(id string, hostConfig *docker.HostConfig) error
	WaitContainer(id string) (int, error)
	KillContainer(opts docker.KillContainerOptions) error
	InspectContainer(id string) (*docker.Container, error)
	AttachToContainer(opts docker.AttachToContainerOptions) error
	RemoveContainer(opts docker.RemoveContainerOptions) error
	Logs(opts docker.LogsOptions) error

	Info() (*docker.Env, error)
}
