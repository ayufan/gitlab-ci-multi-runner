package docker

import (
	"crypto/md5"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsouza/go-dockerclient"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/executors"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
)

type DockerExecutor struct {
	executors.AbstractExecutor
	client    *docker.Client
	image     *docker.Image
	container *docker.Container
	services  []*docker.Container
}

func (s *DockerExecutor) getDockerImage(imageName string) (*docker.Image, error) {
	s.Debugln("Looking for image", imageName, "...")
	image, err := s.client.InspectImage(imageName)
	if err == nil {
		return image, nil
	}

	s.Println("Pulling docker image", imageName, "...")
	pullImageOptions := docker.PullImageOptions{
		Repository: imageName,
		Registry:   helpers.StringOrDefault(s.Config.Docker.Registry, ""),
	}

	err = s.client.PullImage(pullImageOptions, docker.AuthConfiguration{})
	if err != nil {
		return nil, err
	}

	return s.client.InspectImage(imageName)
}

func (s *DockerExecutor) getAbsoluteContainerPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	} else {
		return filepath.Join(s.Build.FullProjectDir(), path)
	}
}

func (s *DockerExecutor) addHostVolume(binds *[]string, hostPath, containerPath string) error {
	containerPath = s.getAbsoluteContainerPath(containerPath)
	s.Debugln("Using host-based", hostPath, "for", containerPath, "...")
	*binds = append(*binds, fmt.Sprintf("%v:%v", hostPath, containerPath))
	return nil
}

func (s *DockerExecutor) addCacheVolume(binds, volumesFrom *[]string, containerPath string) error {
	containerPath = s.getAbsoluteContainerPath(containerPath)

	// disable cache for automatic container cache, but leave it for host volumes (they are shared on purpose)
	if helpers.BoolOrDefault(s.Config.Docker.DisableCache, false) {
		s.Debugln("Container cache for", containerPath, " is disabled.")
		return nil
	}

	hash := md5.Sum([]byte(containerPath))

	// use host-based cache
	if cacheDir := helpers.StringOrDefault(s.Config.Docker.CacheDir, ""); cacheDir != "" {
		hostPath := fmt.Sprintf("%s/%s/%x", s.Config.Docker.CacheDir, s.Build.ProjectUniqueName(), hash)
		hostPath, err := filepath.Abs(hostPath)
		if err != nil {
			return err
		}
		s.Debugln("Using path", hostPath, "as cache for", containerPath, "...")
		*binds = append(*binds, fmt.Sprintf("%v:%v", hostPath, containerPath))
		return nil
	}

	// get existing cache container
	containerName := fmt.Sprintf("%s-cache-%x", s.Build.ProjectUniqueName(), hash)
	container, _ := s.client.InspectContainer(containerName)

	// check if we have valid cache, if not remove the broken container
	if container != nil && container.Volumes[containerPath] == "" {
		s.removeContainer(container.ID)
		container = nil
	}

	// create new cache container for that project
	if container == nil {
		// get busybox image
		cacheImage, err := s.getDockerImage("busybox:latest")
		if err != nil {
			return err
		}

		createContainerOptions := docker.CreateContainerOptions{
			Name: containerName,
			Config: &docker.Config{
				Image: cacheImage.ID,
				Cmd: []string{
					"/bin/true",
				},
				Volumes: map[string]struct{}{
					containerPath: {},
				},
			},
			HostConfig: &docker.HostConfig{},
		}

		container, err = s.client.CreateContainer(createContainerOptions)
		if err != nil {
			if container != nil {
				go s.removeContainer(container.ID)
			}
			return err
		}
	}

	s.Debugln("Using container", container.ID, "as cache", containerPath, "...")
	*volumesFrom = append(*volumesFrom, container.ID)
	return nil
}

func (s *DockerExecutor) addVolume(binds, volumesFrom *[]string, volume string) error {
	var err error
	hostVolume := strings.SplitN(volume, ":", 2)
	switch len(hostVolume) {
	case 2:
		err = s.addHostVolume(binds, hostVolume[0], hostVolume[1])

	case 1:
		// disable cache disables
		err = s.addCacheVolume(binds, volumesFrom, hostVolume[0])
	}

	if err != nil {
		s.Errorln("Failed to create container volume for", volume, err)
	}
	return err
}

func (s *DockerExecutor) createVolumes(image *docker.Image, projectPath string) ([]string, []string, error) {
	var binds, volumesFrom []string

	for _, volume := range s.Config.Docker.Volumes {
		s.addVolume(&binds, &volumesFrom, volume)
	}

	if image != nil {
		for volume := range image.Config.Volumes {
			s.addVolume(&binds, &volumesFrom, volume)
		}
	}

	if s.Build.AllowGitFetch {
		// take path of the projects directory,
		// because we use `rm -rf` which could remove the mounted volume
		s.addVolume(&binds, &volumesFrom, filepath.Dir(projectPath))
	}

	return binds, volumesFrom, nil
}

func (s *DockerExecutor) splitServiceAndVersion(serviceDescription string) (string, string, string) {
	splits := strings.SplitN(serviceDescription, ":", 2)
	service := ""
	version := "latest"
	switch len(splits) {
	case 1:
		service = splits[0]

	case 2:
		service = splits[0]
		version = splits[1]

	default:
		return "", "", ""
	}

	linkName := strings.Replace(service, "/", "__", -1)
	return service, version, linkName
}

func (s *DockerExecutor) createService(service, version string) (*docker.Container, error) {
	if len(service) == 0 {
		return nil, errors.New("invalid service name")
	}

	serviceImage, err := s.getDockerImage(service + ":" + version)
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

func (s *DockerExecutor) getServiceNames() ([]string, error) {
	services := s.Config.Docker.Services

	if servicesOption, ok := s.Build.Options["services"].([]interface{}); ok {
		for _, service := range servicesOption {
			serviceName, ok := service.(string)
			if !ok {
				s.Errorln("Invalid service name passed:", service)
				return nil, errors.New("invalid service name")
			}

			err := s.verifyAllowedImage(serviceName, "services", s.Config.Docker.AllowedServices...)
			if err != nil {
				return nil, err
			}

			services = append(services, serviceName)
		}
	}

	return services, nil
}

func (s *DockerExecutor) createServices() ([]string, error) {
	serviceNames, err := s.getServiceNames()
	if err != nil {
		return nil, err
	}

	linksMap := make(map[string]*docker.Container)

	for _, serviceDescription := range serviceNames {
		service, version, linkName := s.splitServiceAndVersion(serviceDescription)
		if linksMap[linkName] != nil {
			s.Warningln("Service", serviceDescription, "is already created. Ignoring.")
			continue
		}

		container, err := s.createService(service, version)
		if err != nil {
			return nil, err
		}

		s.Debugln("Created service", serviceDescription, "as", container.ID)
		linksMap[linkName] = container
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

	var links []string
	for linkName, container := range linksMap {
		links = append(links, container.ID + ":" + linkName)
	}

	return links, nil
}

func (s *DockerExecutor) connect() (*docker.Client, error) {
	endpoint := "unix:///var/run/docker.sock"
	tlsVerify := false
	tlsCertPath := ""

	if host := helpers.StringOrDefault(s.Config.Docker.Host, ""); host != "" {
		// read docker config from config
		endpoint = host
		if s.Config.Docker.CertPath != nil {
			tlsVerify = true
			tlsCertPath = *s.Config.Docker.CertPath
		}
	} else if host := os.Getenv("DOCKER_HOST"); host != "" {
		// read docker config from environment
		endpoint = host
		tlsVerify, _ = strconv.ParseBool(os.Getenv("DOCKER_TLS_VERIFY"))
		tlsCertPath = os.Getenv("DOCKER_CERT_PATH")
	}

	if tlsVerify {
		client, err := docker.NewTLSClient(
			endpoint,
			filepath.Join(tlsCertPath, "cert.pem"),
			filepath.Join(tlsCertPath, "key.pem"),
			filepath.Join(tlsCertPath, "ca.pem"),
		)
		if err != nil {
			return nil, err
		}

		return client, nil
	} else {
		client, err := docker.NewClient(endpoint)
		if err != nil {
			return nil, err
		}

		return client, nil
	}
}

func (s *DockerExecutor) createContainer(image *docker.Image, cmd []string) (*docker.Container, error) {
	hostname := helpers.StringOrDefault(s.Config.Docker.Hostname, s.Build.ProjectUniqueName())
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

	s.Debugln("Creating cache directories...")
	binds, volumesFrom, err := s.createVolumes(image, s.Build.FullProjectDir())
	if err != nil {
		return nil, err
	}
	createContainerOptions.HostConfig.Binds = binds
	createContainerOptions.HostConfig.VolumesFrom = volumesFrom

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

func (s *DockerExecutor) verifyAllowedImage(image, optionName string, allowedImages... string) error {
	for _, allowedImage := range allowedImages {
		ok, _ := filepath.Match(allowedImage, image)
		if ok {
			return nil
		}
	}

	s.Println()
	if len(allowedImages) != 0 {
		s.Errorln("The", image, "is not present on list of allowed", optionName)
		for _, allowedImage := range allowedImages {
			s.Println("-", allowedImage)
		}
		s.Println()
	} else {
		s.Errorln("No", optionName, "are allowed")
	}

	s.Println("Please check runner's configuration: http://doc.gitlab.com/ci/builds_configuration/docker.html#overwrite-image-and-services")
	return errors.New("invalid image")
}

func (s *DockerExecutor) getImageName() (string, error) {
	if imageOption, ok := s.Build.Options["image"].(string); ok && imageOption != "" {
		err := s.verifyAllowedImage(imageOption, "images", s.Config.Docker.AllowedImages...)
		if err != nil {
			return "", err
		}
		return imageOption, nil
	}

	return s.Config.Docker.Image, nil
}

func (s *DockerExecutor) Prepare(config *common.RunnerConfig, build *common.Build) error {
	err := s.AbstractExecutor.Prepare(config, build)
	if err != nil {
		return err
	}

	if s.ShellScript.PassFile {
		return errors.New("Docker doesn't support shells that require script file")
	}

	if config.Docker == nil {
		return errors.New("Missing docker configuration")
	}

	imageName, err := s.getImageName()
	if err != nil {
		return err
	}

	s.Println("Using Docker executor with image", imageName, "...")

	client, err := s.connect()
	if err != nil {
		return err
	}
	s.client = client

	image, err := s.getDockerImage(imageName)
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
	waitImage, err := s.getDockerImage("aanand/wait")
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
