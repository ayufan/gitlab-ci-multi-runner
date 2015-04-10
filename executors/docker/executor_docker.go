package docker

import (
	"crypto/md5"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsouza/go-dockerclient"

	"github.com/ayufan/gitlab-ci-multi-runner/common"
	"github.com/ayufan/gitlab-ci-multi-runner/executors"
)

type DockerExecutor struct {
	executors.AbstractExecutor
	client    *docker.Client
	image     *docker.Image
	container *docker.Container
	services  []*docker.Container
}

func (s *DockerExecutor) volumeDir(cacheDir string, projectName string, volume string) string {
	hash := md5.Sum([]byte(volume))
	return fmt.Sprintf("%s/%s/%x", cacheDir, projectName, hash)
}

func (s *DockerExecutor) getImage(imageName string, pullImage bool) (*docker.Image, error) {
	s.Debugln("Looking for image", imageName, "...")
	image, err := s.client.InspectImage(imageName)
	if err == nil {
		return image, nil
	}

	if !pullImage {
		return nil, err
	}

	s.Println("Pulling docker image", imageName, "...")
	pullImageOptions := docker.PullImageOptions{
		Repository: imageName,
		Registry:   s.Config.Docker.Registry,
	}

	err = s.client.PullImage(pullImageOptions, docker.AuthConfiguration{})
	if err != nil {
		return nil, err
	}

	return s.client.InspectImage(imageName)
}

func (s *DockerExecutor) addVolume(binds *[]string, cacheDir string, volume string) {
	volumeDir := s.volumeDir(cacheDir, s.Build.ProjectUniqueName(), volume)
	*binds = append(*binds, fmt.Sprintf("%s:%s:rw", volumeDir, volume))
	s.Debugln("Using", volumeDir, "for", volume, "...")

	// TODO: this is potentially insecure
	os.MkdirAll(volumeDir, 0777)
}

func (s *DockerExecutor) createVolumes(image *docker.Image, buildsDir string) ([]string, error) {
	cacheDir := "tmp/docker-cache"
	if len(s.Config.Docker.CacheDir) != 0 {
		cacheDir = s.Config.Docker.CacheDir
	}

	cacheDir, err := filepath.Abs(cacheDir)
	if err != nil {
		return nil, err
	}

	var binds []string

	for _, volume := range s.Config.Docker.Volumes {
		s.addVolume(&binds, cacheDir, volume)
	}

	if image != nil {
		for volume := range image.Config.Volumes {
			s.addVolume(&binds, cacheDir, volume)
		}
	}

	if s.Build.AllowGitFetch {
		s.addVolume(&binds, cacheDir, buildsDir)
	}

	return binds, nil
}

func (s *DockerExecutor) splitServiceAndVersion(service string) (string, string) {
	splits := strings.SplitN(service, ":", 2)
	switch len(splits) {
	case 1:
		return splits[0], "latest"

	case 2:
		return splits[0], splits[1]

	default:
		return "", ""
	}
}

func (s *DockerExecutor) createService(service, version string) (*docker.Container, error) {
	if len(service) == 0 {
		return nil, errors.New("Invalid service name")
	}

	serviceImage, err := s.getImage(service+":"+version, !s.Config.Docker.DisablePull)
	if err != nil {
		return nil, err
	}

	containerName := s.Build.ProjectUniqueName() + "-" + strings.Replace(service, "/", "__", -1)

	// this will fail potentially some builds if there's name collision
	s.removeContainer(containerName)

	s.Println("Starting service", service+":"+version, "...")
	createContainerOpts := docker.CreateContainerOptions{
		Name: containerName,
		Config: &docker.Config{
			Image: serviceImage.ID,
			Env:   s.Config.Environment,
		},
		HostConfig: &docker.HostConfig{
			RestartPolicy: docker.NeverRestart(),
		},
	}

	s.Debugln("Creating service container", createContainerOpts.Name, "...")
	container, err := s.client.CreateContainer(createContainerOpts)
	if err != nil {
		return nil, err
	}

	s.Debugln("Starting service container", container.ID, "...")
	err = s.client.StartContainer(container.ID, createContainerOpts.HostConfig)
	if err != nil {
		go s.removeContainer(container.ID)
		return nil, err
	}

	return container, nil
}

func (s *DockerExecutor) createServices() ([]string, error) {
	var links []string

	for _, serviceDescription := range s.Config.Docker.Services {
		service, version := s.splitServiceAndVersion(serviceDescription)
		container, err := s.createService(service, version)
		if err != nil {
			return links, err
		}

		s.Debugln("Created service", service, version, "as", container.ID)
		links = append(links, container.Name+":"+strings.Replace(service, "/", "__", -1))
		s.services = append(s.services, container)
	}

	waitForServicesTimeout := common.DefaultWaitForServicesTimeout
	if s.Config.Docker.WaitForServicesTimeout != nil {
		waitForServicesTimeout = *s.Config.Docker.WaitForServicesTimeout
	}

	// wait for all services to came up
	if waitForServicesTimeout > 0 && len(s.services) > 0 {
		s.Println("Waiting for services to be up and running...")
		wg := sync.WaitGroup{}
		for _, service := range s.services {
			wg.Add(1)
			go func(service *docker.Container) {
				s.waitForServiceContainer(service, time.Duration(waitForServicesTimeout)*time.Second)
				wg.Done()
			}(service)
		}
		wg.Wait()
	}

	return links, nil
}

func (s *DockerExecutor) connect() (*docker.Client, error) {
	endpoint := s.Config.Docker.Host
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
	hostname := s.Config.Docker.Hostname
	if hostname == "" {
		hostname = s.Build.ProjectUniqueName()
	}

	containerName := s.Build.ProjectUniqueName()

	// this will fail potentially some builds if there's name collision
	s.removeContainer(containerName)

	createContainerOptions := docker.CreateContainerOptions{
		Name: containerName,
		Config: &docker.Config{
			Hostname:     hostname,
			Image:        image.ID,
			Tty:          false,
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			OpenStdin:    true,
			StdinOnce:    true,
			Env:          append(s.ShellScript.Environment, s.Config.Environment...),
			Cmd:          cmd,
		},
		HostConfig: &docker.HostConfig{
			Privileged:    s.Config.Docker.Privileged,
			RestartPolicy: docker.NeverRestart(),
			ExtraHosts:    s.Config.Docker.ExtraHosts,
			Links:         s.Config.Docker.Links,
		},
	}

	s.Debugln("Creating services...")
	links, err := s.createServices()
	if err != nil {
		return nil, err
	}
	createContainerOptions.HostConfig.Links = append(createContainerOptions.HostConfig.Links, links...)

	if !s.Config.Docker.DisableCache {
		s.Debugln("Creating cache directories...")
		binds, err := s.createVolumes(image, s.Build.BuildsDir)
		if err != nil {
			return nil, err
		}
		createContainerOptions.HostConfig.Binds = binds
	}

	s.Debugln("Creating container", createContainerOptions.Name, "...")
	container, err := s.client.CreateContainer(createContainerOptions)
	if err != nil {
		if container != nil {
			go s.removeContainer(container.ID)
		}
		return nil, err
	}

	s.Debugln("Starting container", container.ID, "...")
	err = s.client.StartContainer(container.ID, createContainerOptions.HostConfig)
	if err != nil {
		go s.removeContainer(container.ID)
		return nil, err
	}

	return container, nil
}

func (s *DockerExecutor) removeContainer(id string) error {
	removeContainerOptions := docker.RemoveContainerOptions{
		ID:            id,
		RemoveVolumes: true,
		Force:         true,
	}
	err := s.client.RemoveContainer(removeContainerOptions)
	s.Debugln("Removed container", id, "with", err)
	return err
}

func (s *DockerExecutor) Prepare(config *common.RunnerConfig, build *common.Build) error {
	err := s.AbstractExecutor.Prepare(config, build)
	if err != nil {
		return err
	}

	if s.ShellScript.PassFile {
		return errors.New("Docker doesn't support shells that require script file")
	}

	s.Println("Using Docker executor with image", s.Config.Docker.Image, "...")

	if config.Docker == nil {
		return errors.New("Missing docker configuration")
	}

	client, err := s.connect()
	if err != nil {
		return err
	}
	s.client = client

	// Get image
	image, err := s.getImage(s.Config.Docker.Image, !s.Config.Docker.DisablePull)
	if err != nil {
		return err
	}
	s.image = image
	return nil
}

func (s *DockerExecutor) Cleanup() {
	for _, service := range s.services {
		s.removeContainer(service.ID)
	}

	if s.container != nil {
		s.removeContainer(s.container.ID)
		s.container = nil
	}

	s.AbstractExecutor.Cleanup()
}

func (s *DockerExecutor) waitForServiceContainer(container *docker.Container, timeout time.Duration) error {
	waitImage, err := s.getImage("aanand/wait", !s.Config.Docker.DisablePull)
	if err != nil {
		return err
	}

	waitContainerOpts := docker.CreateContainerOptions{
		Config: &docker.Config{
			Image: waitImage.ID,
		},
		HostConfig: &docker.HostConfig{
			RestartPolicy: docker.NeverRestart(),
			Links:         []string{container.Name + ":" + container.Name},
		},
	}
	s.Debugln("Waiting for service container", container.Name, "to be up and running...")
	waitContainer, err := s.client.CreateContainer(waitContainerOpts)
	if err != nil {
		return err
	}
	defer s.removeContainer(waitContainer.ID)
	err = s.client.StartContainer(waitContainer.ID, waitContainerOpts.HostConfig)
	if err != nil {
		return err
	}

	waitResult := make(chan error, 1)
	go func() {
		statusCode, err := s.client.WaitContainer(waitContainer.ID)
		if err == nil && statusCode != 0 {
			err = fmt.Errorf("Status code: %d", statusCode)
		}
		waitResult <- err
	}()

	// these are warnings and they don't make the build fail
	select {
	case err := <-waitResult:
		if err != nil {
			s.Println("Service", container.Name, "probably didn't start properly", err)
		}
	case <-time.After(timeout):
		s.Println("Service", container.Name, "didn't respond in timely maner:", timeout, "Consider modifying wait_for_services_timeout.")
	}
	return nil
}
