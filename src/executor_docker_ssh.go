package src

import (
	"bytes"
	"errors"
	"io"
	"time"

	"code.google.com/p/go.crypto/ssh"
)

type DockerSshExecutor struct {
	DockerExecutor
	sshClient  *ssh.Client
	sshSession *ssh.Session
}

func (s *DockerSshExecutor) getSshAuthMethods() []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	if len(s.config.Ssh.Password) != 0 {
		methods = append(methods, ssh.Password(s.config.Ssh.Password))
	}

	return methods
}

func (s *DockerSshExecutor) Start() error {
	if s.config.Ssh == nil {
		return errors.New("Missing SSH configuration")
	}

	s.println("Starting SSH command...")

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
		User: s.config.Ssh.User,
		Auth: s.getSshAuthMethods(),
	}

	ssh_host := s.config.Ssh.Host
	if len(ssh_host) == 0 {
		ssh_host = container_data.NetworkSettings.IPAddress
	}

	ssh_port := s.config.Ssh.Port
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

	s.debugln("Creating SSH session...")
	ssh_session, err := ssh_connection.NewSession()
	if err != nil {
		return err
	}
	s.sshSession = ssh_session

	// Setup environment variables
	var envVariables bytes.Buffer
	for _, keyValue := range append(s.build.GetEnv(), s.config.Environment...) {
		envVariables.WriteString("export " + ShellEscape(keyValue) + "\n")
	}

	buffer := io.MultiReader(
		&envVariables,
		bytes.NewBuffer(s.script_data),
	)

	// Wait for process to exit
	go func() {
		s.debugln("Will run SSH command...")
		ssh_session.Stdin = buffer
		ssh_session.Stdout = s.BuildLog
		ssh_session.Stderr = s.BuildLog
		err := ssh_session.Run("bash")
		s.debugln("SSH command finished with", err)
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
