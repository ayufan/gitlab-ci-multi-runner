package src

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"

	"code.google.com/p/go.crypto/ssh"
	"github.com/fsouza/go-dockerclient"
)

type DockerExecutor struct {
	AbstractExecutor
	client    *docker.Client
	image     *docker.Image
	container *docker.Container
}

func (s *DockerExecutor) volumeDir(cache_dir string, project_name string, volume string) string {
	hash := md5.Sum([]byte(volume))
	return fmt.Sprintf("%s/%s/%x", cache_dir, project_name, hash)
}

func (s *DockerExecutor) getImage(imageName string, pullImage bool) (*docker.Image, error) {
	s.debugln("Looking for image", imageName, "...")
	image, err := s.client.InspectImage(imageName)
	if err == nil {
		return image, nil
	}

	if !pullImage {
		return nil, err
	}

	s.println("Pulling docker image", imageName, "...")
	pull_image_opts := docker.PullImageOptions{
		Repository: imageName,
		Registry:   s.config.DockerRegistry,
	}

	err = s.client.PullImage(pull_image_opts, docker.AuthConfiguration{})
	if err != nil {
		return nil, err
	}

	return s.client.InspectImage(imageName)
}

func (s *DockerExecutor) addVolume(binds *[]string, cache_dir string, volume string) {
	volumeDir := s.volumeDir(cache_dir, s.build.ProjectUniqueName(), volume)
	*binds = append(*binds, fmt.Sprintf("%s:%s:rw", volumeDir, volume))
	s.debugln("Using", volumeDir, "for", volume, "...")

	// TODO: this is potentially insecure
	os.MkdirAll(volumeDir, 0777)
}

func (s *DockerExecutor) createVolumes(image *docker.Image, builds_dir string) ([]string, error) {
	cache_dir := "tmp/docker-cache"
	if len(s.config.DockerCacheDir) != 0 {
		cache_dir = s.config.DockerCacheDir
	}

	cache_dir, err := filepath.Abs(cache_dir)
	if err != nil {
		return nil, err
	}

	var binds []string

	for _, volume := range s.config.DockerVolumes {
		s.addVolume(&binds, cache_dir, volume)
	}

	if image != nil {
		for volume, _ := range image.Config.Volumes {
			s.addVolume(&binds, cache_dir, volume)
		}
	}

	if s.build.AllowGitFetch {
		s.addVolume(&binds, cache_dir, builds_dir)
	}

	return binds, nil
}

func (s *DockerExecutor) connect() (*docker.Client, error) {
	endpoint := s.config.DockerHost
	if len(endpoint) == 0 {
		endpoint = os.Getenv("DOCKER_HOST")
	}
	if len(endpoint) == 0 {
		endpoint = "unix:///var/run/docker.sock"
	}
	client, err := docker.NewClient(endpoint)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (s *DockerExecutor) createContainer(image *docker.Image, cmd []string) (*docker.Container, error) {
	create_container_opts := docker.CreateContainerOptions{
		Name: s.build.ProjectUniqueName(),
		Config: &docker.Config{
			Hostname:     s.build.ProjectUniqueName(),
			Image:        image.ID,
			Tty:          false,
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			OpenStdin:    true,
			StdinOnce:    true,
			Env:          append(s.build.GetEnv(), s.config.Environment...),
			Cmd:          cmd,
		},
		HostConfig: &docker.HostConfig{
			Privileged:    s.config.DockerPrivileged,
			RestartPolicy: docker.NeverRestart(),
			ExtraHosts:    s.config.DockerExtraHosts,
			Links:         s.config.DockerLinks,
		},
	}

	if !s.config.DockerDisableCache {
		s.debugln("Creating cache directories...")
		binds, err := s.createVolumes(image, s.builds_dir)
		if err != nil {
			return nil, err
		}
		create_container_opts.HostConfig.Binds = binds
	}

	s.debugln("Creating contaier...")
	container, err := s.client.CreateContainer(create_container_opts)
	if err != nil {
		return nil, err
	}

	s.debugln("Starting container", container.ID, "...")
	err = s.client.StartContainer(container.ID, create_container_opts.HostConfig)
	if err != nil {
		go s.removeContainer(container.ID)
		return nil, err
	}

	return container, nil
}

func (s *DockerExecutor) removeContainer(id string) {
	remove_container_opts := docker.RemoveContainerOptions{
		ID:            id,
		RemoveVolumes: true,
		Force:         true,
	}
	err := s.client.RemoveContainer(remove_container_opts)
	s.debugln("Removed container", id, "with", err)
}

func (s *DockerExecutor) getSshAuthMethods() []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	if len(s.config.SshPassword) != 0 {
		methods = append(methods, ssh.Password(s.config.SshPassword))
	}

	return methods
}

func (s *DockerExecutor) Prepare(config *RunnerConfig, build *Build) error {
	err := s.AbstractExecutor.Prepare(config, build)
	if err != nil {
		return err
	}

	s.println("Using Docker executor with image", s.config.DockerImage, "...")

	client, err := s.connect()
	if err != nil {
		return err
	}
	s.client = client

	// Get image
	image, err := s.getImage(s.config.DockerImage, !s.config.DockerDisablePull)
	if err != nil {
		return err
	}
	s.image = image
	return nil
}

func (s *DockerExecutor) Cleanup() {
	if s.container != nil {
		s.removeContainer(s.container.ID)
		s.container = nil
	}

	s.AbstractExecutor.Cleanup()
}
