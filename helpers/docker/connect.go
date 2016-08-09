package docker_helpers

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
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

type cacheKey struct {
	credentials DockerCredentials
	apiVersion  string
}

var (
	cache   = make(map[cacheKey]Client)
	cacheMu sync.Mutex
)

func New(c DockerCredentials, apiVersion string) (Client, error) {
	key := cacheKey{credentials: c, apiVersion: apiVersion}

	cacheMu.Lock()
	defer cacheMu.Unlock()

	if c, ok := cache[key]; ok {
		return c, nil
	}

	client, err := build(c, apiVersion)
	if err == nil {
		cache[key] = client
	}

	return client, err
}

func build(c DockerCredentials, apiVersion string) (client Client, err error) {
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
