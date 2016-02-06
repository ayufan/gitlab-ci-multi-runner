package docker_helpers

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/fsouza/go-dockerclient"
)

func Connect(c DockerCredentials, apiVersion string) (*docker.Client, error) {
	endpoint := "unix:///var/run/docker.sock"
	tlsVerify := false
	tlsCertPath := ""

	if c.Host != "" {
		// read docker config from config
		endpoint = c.Host
		if c.CertPath != "" {
			tlsVerify = true
			tlsCertPath = c.CertPath
		}
	} else if host := os.Getenv("DOCKER_HOST"); host != "" {
		// read docker config from environment
		endpoint = host
		tlsVerify, _ = strconv.ParseBool(os.Getenv("DOCKER_TLS_VERIFY"))
		tlsCertPath = os.Getenv("DOCKER_CERT_PATH")
	}

	if tlsVerify {
		client, err := docker.NewVersionnedTLSClient(
			endpoint,
			filepath.Join(tlsCertPath, "cert.pem"),
			filepath.Join(tlsCertPath, "key.pem"),
			filepath.Join(tlsCertPath, "ca.pem"),
			apiVersion,
		)
		if err != nil {
			return nil, err
		}

		return client, nil
	}

	client, err := docker.NewVersionedClient(endpoint, apiVersion)
	if err != nil {
		return nil, err
	}

	return client, nil
}
