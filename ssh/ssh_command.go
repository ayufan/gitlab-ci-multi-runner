package ssh

import (
	"bytes"
	"errors"
	"io"
	"time"

	"code.google.com/p/go.crypto/ssh"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
)

type Command struct {
	Config

	Environment []string
	Command     string
	Stdin       []byte
	Stdout      io.Writer
	Stderr      io.Writer

	ConnectRetries int

	client *ssh.Client
}

func (s *Command) getSSHAuthMethods() []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	if s.Password != nil {
		methods = append(methods, ssh.Password(*s.Password))
	}

	return methods
}

func (s *Command) Connect() error {
	host := helpers.StringOrDefault(s.Host, "localhost")
	user := helpers.StringOrDefault(s.User, "root")
	port := helpers.StringOrDefault(s.Port, "22")

	config := &ssh.ClientConfig{
		User: user,
		Auth: s.getSSHAuthMethods(),
	}

	connectRetries := s.ConnectRetries
	if connectRetries == 0 {
		connectRetries = 3
	}

	var finalError error

	for i := 0; i < connectRetries; i++ {
		client, err := ssh.Dial("tcp", host+":"+port, config)
		if err == nil {
			s.client = client
			return nil
		}
		time.Sleep(sshRetryInterval * time.Second)
		finalError = err
	}

	return finalError
}

func (s *Command) Exec(cmd string) error {
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

func (s *Command) Run() error {
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

func (s *Command) Cleanup() {
	if s.client != nil {
		s.client.Close()
	}
}
