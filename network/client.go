package network

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

type client struct {
	http.Client
	url        *url.URL
	caFile     string
	skipVerify bool
	updateTime time.Time
}

func (n *client) ensureTlsConfig() {
	// certificate got modified
	if stat, err := os.Stat(n.caFile); err == nil && n.updateTime.Before(stat.ModTime()) {
		n.Transport = nil
	}

	// create or update transport
	if n.Transport == nil {
		n.updateTime = time.Now()
		n.createTransport()
	}
}

func (n *client) createTransport() {
	// create reference TLS config
	tlsConfig := tls.Config{
		MinVersion:         tls.VersionTLS10,
		InsecureSkipVerify: n.skipVerify,
	}

	// load TLS certificate
	if file := n.caFile; file != "" && !n.skipVerify {
		logrus.Debugln("Trying to load", file, "...")

		data, err := ioutil.ReadFile(file)
		if err == nil {
			pool := x509.NewCertPool()
			if pool.AppendCertsFromPEM(data) {
				tlsConfig.RootCAs = pool
			} else {
				logrus.Errorln("Failed to parse PEM in", n.caFile)
			}
		} else {
			if !os.IsNotExist(err) {
				logrus.Errorln("Failed to load", n.caFile, err)
			}
		}
	}

	// create transport
	n.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     &tlsConfig,
	}
}

type RequestPreparer func(uri string) (*http.Request, error)

func (n *client) do(uri string, prepRequest RequestPreparer, statusCode int, response interface{}) (int, string) {
	url, err := n.url.Parse(uri)
	if err != nil {
		return -1, err.Error()
	}

	req, err := prepRequest(url.String())
	if err != nil {
		return -1, fmt.Sprintf("failed to prepare request: %v", err)
	}

	if response != nil {
		req.Header.Set("Accept", "application/json")
	}

	n.ensureTlsConfig()

	res, err := n.Do(req)
	if err != nil {
		return -1, fmt.Sprintf("couldn't execute %v against %s: %v", req.Method, req.URL, err)
	}
	defer res.Body.Close()

	if res.StatusCode == statusCode {
		if response != nil {
			if contentType := res.Header.Get("Content-Type"); contentType != "application/json" {
				return -1, fmt.Sprintf("Server should return application/json. Got: %v", contentType)
			}

			d := json.NewDecoder(res.Body)
			err = d.Decode(response)
			if err != nil {
				return -1, fmt.Sprintf("Error decoding json payload %v", err)
			}
		}
	}

	return res.StatusCode, res.Status
}

func (n *client) doJson(uri, method string, statusCode int, request interface{}, response interface{}) (int, string) {
	return n.do(uri, func(url string) (*http.Request, error) {
		var body []byte
		var err error
		if request != nil {
			body, err = json.Marshal(request)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal project object: %v", err)
			}
		}

		req, err := http.NewRequest(method, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}

		if request != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		return req, nil
	}, statusCode, response)
}

func (n *client) fullUrl(uri string, a ...interface{}) string {
	url, err := n.url.Parse(fmt.Sprintf(uri, a...))
	if err != nil {
		return ""
	}
	return url.String()
}

func newClient(config common.RunnerCredentials) (c *client, err error) {
	url, err := url.Parse(strings.TrimRight(config.URL, "/") + "/api/v1/")
	if err != nil {
		return
	}

	if url.Scheme != "http" && url.Scheme != "https" {
		err = errors.New("only http or https scheme supported")
		return
	}

	c = &client{
		url:        url,
		skipVerify: config.TLSSkipVerify,
		caFile:     config.TLSCAFile,
	}

	if CertificateDirectory != "" && c.caFile == "" {
		hostAndPort := strings.Split(url.Host, ":")
		c.caFile = filepath.Join(CertificateDirectory, hostAndPort[0]+".crt")
	}

	return
}
