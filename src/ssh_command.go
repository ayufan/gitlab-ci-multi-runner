package src

import (
	"bytes"
	"errors"
	"io"
	"time"

	"code.google.com/p/go.crypto/ssh"
)

type SshCommand struct {
	SshConfig

	Environment []string
	Command     string
	Stdin       []byte
	Stdout      io.Writer
	Stderr      io.Writer

	client  *ssh.Client
	session *ssh.Session
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

	client, err := ssh.Dial("tcp", s.Host+":"+s.Port, config)
	if err != nil {
		for i := 0; i < 3 && err != nil; i++ {
			time.Sleep(SSH_RETRY_INTERVAL * time.Second)
			client, err = ssh.Dial("tcp", s.Host+":"+s.Port, config)
		}
		if err != nil {
			return err
		}
	}
	s.client = client

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	s.session = session
	return nil
}

func (s *SshCommand) Run() error {
	if s.session == nil {
		return errors.New("Not connected")
	}

	var envVariables bytes.Buffer
	for _, keyValue := range s.Environment {
		envVariables.WriteString("export " + ShellEscape(keyValue) + "\n")
	}

	s.session.Stdin = io.MultiReader(
		&envVariables,
		bytes.NewBuffer(s.Stdin),
	)
	s.session.Stdout = s.Stdout
	s.session.Stderr = s.Stderr
	err := s.session.Run(s.Command)
	return err
}

func (s *SshCommand) Cleanup() {
	if s.session != nil {
		s.session.Close()
	}
	if s.client != nil {
		s.client.Close()
	}
}
