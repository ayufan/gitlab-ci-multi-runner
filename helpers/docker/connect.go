package docker_helpers

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/helpers"
)

var dockerDialer = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
}

func httpTransportFix(host string, client *docker.Client) {
	logrus.WithField("host", host).Debugln("Applying docker.Client transport fix:", client)
	client.Dialer = dockerDialer
	client.HTTPClient = &http.Client{
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			Dial:                dockerDialer.Dial,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig:     client.TLSConfig,
		},
	}
}

func New(c DockerCredentials, apiVersion string) (client *docker.Client, err error) {
	endpoint := "unix:///var/run/docker.sock"
	tlsVerify := false
	tlsCertPath := ""

	defer func() {
		if client != nil {
			httpTransportFix(endpoint, client)
		}
	}()

	if host := helpers.StringOrDefault(c.Host, ""); host != "" {
		// read docker config from config
		endpoint = host
		if c.CertPath != nil {
			tlsVerify = true
			tlsCertPath = *c.CertPath
		}
	} else if host := os.Getenv("DOCKER_HOST"); host != "" {
		// read docker config from environment
		endpoint = host
		tlsVerify, _ = strconv.ParseBool(os.Getenv("DOCKER_TLS_VERIFY"))
		tlsCertPath = os.Getenv("DOCKER_CERT_PATH")
	}

	if tlsVerify {
		client, err = docker.NewVersionnedTLSClient(
			endpoint,
			filepath.Join(tlsCertPath, "cert.pem"),
			filepath.Join(tlsCertPath, "key.pem"),
			filepath.Join(tlsCertPath, "ca.pem"),
			apiVersion,
		)
		return
	}

	client, err = docker.NewVersionedClient(endpoint, apiVersion)
	return
}

func Close(client *docker.Client) {
	// Nuke all connections
	if transport, ok := client.HTTPClient.Transport.(*http.Transport); ok && transport != http.DefaultTransport {
		transport.DisableKeepAlives = true
		transport.CloseIdleConnections()
		logrus.Debugln("Closed all idle connections for docker.Client:", client)
	}
}
