package ssh

import (
	"bytes"
	"errors"
	"io"
	"time"

	"golang.org/x/crypto/ssh"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
	"io/ioutil"
	"strings"
)

type Client struct {
	Config

	Stdout         io.Writer
	Stderr         io.Writer
	ConnectRetries int

	client *ssh.Client
}

type Command struct {
	Environment []string
	Command     []string
	Stdin       string
	Abort       chan interface{}
}

func (s *Client) getSSHKey(identityFile string) (key ssh.Signer, err error) {
	buf, err := ioutil.ReadFile(identityFile)
	if err != nil {
		return nil, err
	}
	key, err = ssh.ParsePrivateKey(buf)
	return key, err
}

func (s *Client) getSSHAuthMethods() ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod
	methods = append(methods, ssh.Password(s.Password))

	if s.IdentityFile != "" {
		key, err := s.getSSHKey(s.IdentityFile)
		if err != nil {
			return nil, err
		}
		methods = append(methods, ssh.PublicKeys(key))
	}

	return methods, nil
}

func (s *Client) Connect() error {
	if s.Host == "" {
		s.Host = "localhost"
	}
	if s.User == "" {
		s.User = "root"
	}
	if s.Port == "" {
		s.Port = "22"
	}

	methods, err := s.getSSHAuthMethods()
	if err != nil {
		return err
	}

	config := &ssh.ClientConfig{
		User: s.User,
		Auth: methods,
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
		time.Sleep(sshRetryInterval * time.Second)
		finalError = err
	}

	return finalError
}

func (s *Client) Exec(cmd string) error {
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

func (s *Command) fullCommand() string {
	var arguments []string
	// TODO: This method is compatible only with Bjourne compatible shells
	for _, part := range s.Command {
		arguments = append(arguments, helpers.ShellEscape(part))
	}
	return strings.Join(arguments, " ")
}

func (s *Client) Run(cmd Command) error {
	if s.client == nil {
		return errors.New("Not connected")
	}

	session, err := s.client.NewSession()
	if err != nil {
		return err
	}

	var envVariables bytes.Buffer
	for _, keyValue := range cmd.Environment {
		envVariables.WriteString("export " + helpers.ShellEscape(keyValue) + "\n")
	}

	session.Stdin = io.MultiReader(
		&envVariables,
		bytes.NewBufferString(cmd.Stdin),
	)
	session.Stdout = s.Stdout
	session.Stderr = s.Stderr
	err = session.Start(cmd.fullCommand())
	if err != nil {
		return err
	}
	defer session.Close()

	waitCh := make(chan error)
	go func() {
		waitCh <- session.Wait()
	}()

	select {
	case <-cmd.Abort:
		session.Signal(ssh.SIGKILL)
		return <-waitCh

	case err := <-waitCh:
		return err
	}
}

func (s *Client) Cleanup() {
	if s.client != nil {
		s.client.Close()
	}
}
