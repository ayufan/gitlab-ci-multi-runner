package src

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"code.google.com/p/go.crypto/ssh"
	"github.com/fsouza/go-dockerclient"
)

type DockerMode string

type DockerExecutor struct {
	BaseExecutor
	client    *docker.Client
	image     *docker.Image
	container *docker.Container
}

func (s *DockerExecutor) volumeDir(cache_dir string, project_name string, volume string) string {
	hash := md5.Sum([]byte(volume))
	return fmt.Sprintf("%s/%s/%x", cache_dir, project_name, hash)
}

func (s *DockerExecutor) getImage(imageName string, pullImage bool) (*docker.Image, error) {
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
	s.debugln("Using", volumeDir, "for", volume)
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
	// Connect to docker
	endpoint := s.config.DockerHost
	if len(endpoint) == 0 {
		endpoint = os.Getenv("DOCKER_HOST")
	}
	if len(endpoint) == 0 {
		return nil, errors.New("No DOCKER_HOST defined")
	}
	client, err := docker.NewClient(endpoint)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (s *DockerExecutor) createContainer(image *docker.Image, cmd []string) (*docker.Container, error) {
	s.debugln("Creating contaier")
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
		s.debugln("Creating cache dirs")
		binds, err := s.createVolumes(image, s.builds_dir)
		if err != nil {
			return nil, err
		}
		create_container_opts.HostConfig.Binds = binds
	}

	container, err := s.client.CreateContainer(create_container_opts)
	if err != nil {
		return nil, err
	}

	s.debugln("Starting container")
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
	s.client.RemoveContainer(remove_container_opts)
}

func (s *DockerExecutor) getSshAuthMethods() []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	if len(s.config.SshPassword) != 0 {
		methods = append(methods, ssh.Password(s.config.SshPassword))
	}

	return methods
}

func (s *DockerExecutor) Prepare(config *RunnerConfig, build *Build) error {
	err := s.BaseExecutor.Prepare(config, build)
	if err != nil {
		return err
	}

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

	s.BaseExecutor.Cleanup()
}

type DockerCommandExecutor struct {
	DockerExecutor
}

func (s *DockerCommandExecutor) Start() error {
	// Create container
	container, err := s.createContainer(s.image, []string{"bash"})
	if err != nil {
		return err
	}
	s.container = container

	// Wait for process to exit
	go func() {
		attach_container_opts := docker.AttachToContainerOptions{
			Container:    container.ID,
			InputStream:  bytes.NewBuffer(s.script_data),
			OutputStream: s.build_log,
			ErrorStream:  s.build_log,
			Logs:         true,
			Stream:       true,
			Stdin:        true,
			Stdout:       true,
			Stderr:       true,
			RawTerminal:  false,
		}

		s.debugln("Attach to container")
		err := s.client.AttachToContainer(attach_container_opts)
		if err != nil {
			s.buildFinish <- err
			return
		}

		s.debugln("Wait for container")
		exit_code, err := s.client.WaitContainer(container.ID)
		if err != nil {
			s.buildFinish <- err
			return
		}

		if exit_code == 0 {
			s.buildFinish <- nil
		} else {
			s.buildFinish <- errors.New(fmt.Sprintf("exit code", exit_code))
		}
	}()
	return nil
}

type DockerSshExecutor struct {
	DockerExecutor
	sshClient  *ssh.Client
	sshSession *ssh.Session
}

func (s *DockerSshExecutor) Start() error {
	// Create container
	container, err := s.createContainer(s.image, []string{})
	if err != nil {
		return err
	}
	s.container = container

	container_data, err := s.client.InspectContainer(container.ID)
	if err != nil {
		return err
	}

	ssh_config := &ssh.ClientConfig{
		User: s.config.SshUser,
		Auth: s.getSshAuthMethods(),
	}

	ssh_host := s.config.SshHost
	if len(ssh_host) == 0 {
		ssh_host = container_data.NetworkSettings.IPAddress
	}

	ssh_port := s.config.SshPort
	if len(ssh_port) == 0 {
		ssh_port = "22"
	}

	s.debugln("Connecting to", ssh_host, ssh_port, "as", ssh_config.User)
	ssh_connection, err := ssh.Dial("tcp", ssh_host+":"+ssh_port, ssh_config)
	if err != nil {
		for i := 0; i < 3 && err != nil; i++ {
			time.Sleep(SSH_RETRY_INTERVAL * time.Second)
			ssh_connection, err = ssh.Dial("tcp", ssh_host+":"+ssh_port, ssh_config)
		}
		if err != nil {
			return err
		}
	}
	s.sshClient = ssh_connection

	s.debugln("Creating new session...")
	ssh_session, err := ssh_connection.NewSession()
	if err != nil {
		return err
	}
	s.sshSession = ssh_session

	// Wait for process to exit
	go func() {
		s.debugln("Running new command...")
		ssh_session.Stdin = bytes.NewBuffer(s.script_data)
		ssh_session.Stdout = s.build_log
		ssh_session.Stderr = s.build_log
		err := ssh_session.Run("bash")
		s.debugln("Ssh command finished with", err)
		s.buildFinish <- err
	}()
	return nil
}

func (s *DockerSshExecutor) Cleanup() {
	if s.sshSession != nil {
		s.sshSession.Close()
	}
	if s.sshClient != nil {
		s.sshClient.Close()
	}

	s.DockerExecutor.Cleanup()
}
