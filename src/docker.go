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
}

type DockerCommandExecutor struct {
	DockerExecutor
}

type DockerSshExecutor struct {
	DockerExecutor
}

func (s *DockerExecutor) volumeDir(cache_dir string, project_name string, volume string) string {
	hash := md5.Sum([]byte(volume))
	return fmt.Sprintf("%s/%s/%x", cache_dir, project_name, hash)
}

func (s *DockerExecutor) getImage(client *docker.Client) (*docker.Image, error) {
	image, err := client.InspectImage(s.config.DockerImage)
	if err == nil {
		return image, nil
	}

	if s.config.DockerDisablePull {
		return nil, err
	}

	s.println("Pulling docker image", s.config.DockerImage, "...")
	pull_image_opts := docker.PullImageOptions{
		Repository: s.config.DockerImage,
		Registry:   s.config.DockerRegistry,
	}

	err = client.PullImage(pull_image_opts, docker.AuthConfiguration{})
	if err != nil {
		return nil, err
	}

	return client.InspectImage(s.config.DockerImage)
}

func (s *DockerExecutor) addVolume(binds *[]string, cache_dir string, volume string) {
	volumeDir := s.volumeDir(cache_dir, s.build.ProjectUniqueName(), volume)
	*binds = append(*binds, fmt.Sprintf("%s:%s:rw", volumeDir, volume))
	s.debugln("Using", volumeDir, "for", volume)
}

func (s *DockerExecutor) createVolumes(client *docker.Client, image *docker.Image, builds_dir string) ([]string, error) {
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

func (s *DockerExecutor) createContainer(client *docker.Client, image *docker.Image, cmd []string) (*docker.Container, error) {
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
		binds, err := s.createVolumes(client, image, s.builds_dir)
		if err != nil {
			return nil, err
		}
		create_container_opts.HostConfig.Binds = binds
	}

	container, err := client.CreateContainer(create_container_opts)
	if err != nil {
		return nil, err
	}

	s.debugln("Starting container")
	err = client.StartContainer(container.ID, create_container_opts.HostConfig)
	if err != nil {
		go s.removeContainer(client, container.ID)
		return nil, err
	}

	return container, nil
}

func (s *DockerExecutor) removeContainer(client *docker.Client, id string) {
	remove_container_opts := docker.RemoveContainerOptions{
		ID:            id,
		RemoveVolumes: true,
		Force:         true,
	}
	client.RemoveContainer(remove_container_opts)
}

func (s *DockerExecutor) getSshAuthMethods() []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	if len(s.config.SshPassword) != 0 {
		methods = append(methods, ssh.Password(s.config.SshPassword))
	}

	return methods
}

func (s *DockerCommandExecutor) Start() error {
	client, err := s.connect()
	if err != nil {
		return err
	}

	// Get image
	image, err := s.getImage(client)
	if err != nil {
		return err
	}

	// Create container
	container, err := s.createContainer(client, image, []string{"bash"})
	if err != nil {
		return err
	}

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
		err := client.AttachToContainer(attach_container_opts)
		if err != nil {
			s.buildFinish <- err
			return
		}

		s.debugln("Wait for container")
		exit_code, err := client.WaitContainer(container.ID)
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

	s.buildAbortFunc = func(e *BaseExecutor) {
		s.removeContainer(client, container.ID)
	}
	return nil
}

func (s *DockerSshExecutor) Start() error {
	client, err := s.connect()
	if err != nil {
		return err
	}

	// Get image
	image, err := s.getImage(client)
	if err != nil {
		return err
	}

	// Create container
	container, err := s.createContainer(client, image, []string{})
	if err != nil {
		return err
	}
	defer s.removeContainer(client, container.ID)

	container_data, err := client.InspectContainer(container.ID)
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

	s.debugln("Creating new session...")
	ssh_session, err := ssh_connection.NewSession()
	if err != nil {
		return err
	}

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

	s.buildAbortFunc = func(e *BaseExecutor) {
		ssh_session.Close()
	}
	return nil
}
