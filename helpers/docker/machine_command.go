package docker_helpers

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/Sirupsen/logrus"
)

type machineCommand struct {
}

func (m *machineCommand) Create(driver, name string, opts ...string) error {
	args := []string{
		"create",
		"--driver", driver,
	}
	for _, opt := range opts {
		keyValue := strings.SplitN(opt, "=", 2)
		if len(keyValue) > 0 {
			args = append(args, "--"+keyValue[0])
		}
		if len(keyValue) > 1 {
			args = append(args, keyValue[1])
		}
	}
	args = append(args, name)

	cmd := exec.Command("docker-machine", args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	logrus.Debugln("Executing", cmd.Path, cmd.Args)
	return cmd.Run()
}

func (m *machineCommand) Provision(name string) error {
	cmd := exec.Command("docker-machine", "provision", name)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m *machineCommand) Remove(name string) error {
	cmd := exec.Command("docker-machine", "rm", "-f", name)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m *machineCommand) List(nodeFilter string) (machines []string, err error) {
	cmd := exec.Command("docker-machine", "ls", "-q")
	cmd.Env = os.Environ()
	data, err := cmd.Output()
	if err != nil {
		return
	}

	reader := bufio.NewReader(bytes.NewReader(data))
	for {
		var line string

		line, err = reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var query string
		if n, _ := fmt.Sscanf(line, nodeFilter, &query); n != 1 {
			continue
		}

		machines = append(machines, line)
	}
}

func (m *machineCommand) get(args ...string) (out string, err error) {
	// Execute docker-machine to fetch IP
	cmd := exec.Command("docker-machine", args...)
	cmd.Env = os.Environ()
	data, err := cmd.Output()
	if err != nil {
		return
	}

	// Save the IP
	out = strings.TrimSpace(string(data))
	if out == "" {
		err = fmt.Errorf("failed to get %v", args)
	}
	return
}

func (m *machineCommand) IP(name string) (string, error) {
	return m.get("ip", name)
}

func (m *machineCommand) URL(name string) (string, error) {
	return m.get("url", name)
}

func (m *machineCommand) CertPath(name string) (string, error) {
	return m.get("inspect", name, "-f", "{{.HostOptions.AuthOptions.StorePath}}")
}

func (m *machineCommand) Status(name string) (string, error) {
	return m.get("status", name)
}

func (m *machineCommand) CanConnect(name string) bool {
	status, err := m.Status(name)
	if err != nil {
		return false
	}
	if status == "Running" {
		return true
	}
	return false
}

func (m *machineCommand) Credentials(name string) (dc DockerCredentials, err error) {
	if !m.CanConnect(name) {
		err = errors.New("Can't connect")
		return
	}

	dc.TLSVerify = true
	dc.Host, err = m.URL(name)
	if err == nil {
		dc.CertPath, err = m.CertPath(name)
	}
	return
}

func NewMachineCommand() Machine {
	return &machineCommand{}
}
