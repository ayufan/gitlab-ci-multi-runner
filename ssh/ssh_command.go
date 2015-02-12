package ssh

import (
	"bytes"
	"errors"
	"io"
	"time"

	"code.google.com/p/go.crypto/ssh"

	"github.com/ayufan/gitlab-ci-multi-runner/helpers"
)

type SshCommand struct {
	SshConfig

	Environment []string
	Command     string
	Stdin       []byte
	Stdout      io.Writer
	Stderr      io.Writer

	ConnectRetries int

	client *ssh.Client
}

func (s *SshCommand) getSshAuthMethods() []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	if len(s.Password) != 0 {
		methods = append(methods, ssh.Password(s.Password))
	}

	return methods
}

func (s *SshCommand) Connect() error {
	if len(s.User) == 0 {
		s.User = "root"
	}
	if len(s.Port) == 0 {
		s.Port = "22"
	}

	config := &ssh.ClientConfig{
		User: s.User,
		Auth: s.getSshAuthMethods(),
	}

	connectRetries := s.ConnectRetries
	if connectRetries == 0 {
		connectRetries = 3
	}

	var finalError error

	for i := 0; i < connectRetries; i++ {
		client, err := ssh.Dial("tcp", s.Host+":"+s.Port, config)
		if err == nil {
			s.client = client
			return nil
		}
		time.Sleep(SSH_RETRY_INTERVAL * time.Second)
		finalError = err
	}

	return finalError
}

func (s *SshCommand) Exec(cmd string) error {
	if s.client == nil {
		return errors.New("Not connected")
	}

	session, err := s.client.NewSession()
	if err != nil {
		return err
	}
	session.Stdout = s.Stdout
	session.Stderr = s.Stderr
	err = session.Run(cmd)
	session.Close()
	return err
}

func (s *SshCommand) Run() error {
	if s.client == nil {
		return errors.New("Not connected")
	}

	session, err := s.client.NewSession()
	if err != nil {
		return err
	}

	var envVariables bytes.Buffer
	for _, keyValue := range s.Environment {
		envVariables.WriteString("export " + helpers.ShellEscape(keyValue) + "\n")
	}

	session.Stdin = io.MultiReader(
		&envVariables,
		bytes.NewBuffer(s.Stdin),
	)
	session.Stdout = s.Stdout
	session.Stderr = s.Stderr
	err = session.Run(s.Command)
	session.Close()
	return err
}

func (s *SshCommand) Cleanup() {
	if s.client != nil {
		s.client.Close()
	}
}
