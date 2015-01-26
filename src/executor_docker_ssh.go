package src

import (
	"bytes"
	"io"
	"time"

	"code.google.com/p/go.crypto/ssh"
)

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
		s.debugln("Running SSH command...")
		ssh_session.Stdin = buffer
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
