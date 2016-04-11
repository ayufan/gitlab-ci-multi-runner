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
)

var dockerDialer = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
}

func httpTransportFix(host string, client Client) {
	dockerClient, ok := client.(*docker.Client)
	if !ok || dockerClient == nil {
		return
	}

	logrus.WithField("host", host).Debugln("Applying docker.Client transport fix:", dockerClient)
	dockerClient.Dialer = dockerDialer
	dockerClient.HTTPClient = &http.Client{
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			Dial:                dockerDialer.Dial,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig:     dockerClient.TLSConfig,
		},
	}
}

func New(c DockerCredentials, apiVersion string) (client Client, err error) {
	endpoint := "unix:///var/run/docker.sock"
	tlsVerify := false
	tlsCertPath := ""

	defer func() {
		if client != nil {
			httpTransportFix(endpoint, client)
		}
	}()

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
		client, err = docker.NewVersionedTLSClient(
			endpoint,
			filepath.Join(tlsCertPath, "cert.pem"),
			filepath.Join(tlsCertPath, "key.pem"),
			filepath.Join(tlsCertPath, "ca.pem"),
			apiVersion,
		)
		if err != nil {
			logrus.Errorln("Error while TLS Docker client creation:", err)
		}
		return
	}

	client, err = docker.NewVersionedClient(endpoint, apiVersion)
	if err != nil {
		logrus.Errorln("Error while Docker client creation:", err)
	}
	return
}

func Close(client Client) {
	dockerClient, ok := client.(*docker.Client)
	if !ok {
		return
	}

	// Nuke all connections
	if transport, ok := dockerClient.HTTPClient.Transport.(*http.Transport); ok && transport != http.DefaultTransport {
		transport.DisableKeepAlives = true
		transport.CloseIdleConnections()
		logrus.Debugln("Closed all idle connections for docker.Client:", dockerClient)
	}
}
