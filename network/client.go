package network

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var dialer = net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
}

type client struct {
	http.Client
	url        *url.URL
	caFile     string
	skipVerify bool
	updateTime time.Time
}

func (n *client) ensureTLSConfig() {
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
		Dial: func(network, addr string) (net.Conn, error) {
			logrus.Debugln("Dialing:", network, addr, "...")
			return dialer.Dial(network, addr)
		},
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     &tlsConfig,
	}
}

func (n *client) getCAChain(tls *tls.ConnectionState) (certificates string) {
	if tls == nil {
		return
	}

	list := make(map[string]*x509.Certificate)

	for _, verifiedChain := range tls.VerifiedChains {
	nextInChain:
		for _, certificate := range verifiedChain {
			signature := hex.EncodeToString(certificate.Signature)
			if list[signature] != nil {
				continue
			}

			// Always add signed by yourself
			if certificate.CheckSignatureFrom(certificate) == nil {
				list[signature] = certificate
				continue
			}

			// We don't need to add certificates that are returned by server
			for _, peerCertificate := range tls.PeerCertificates {
				if peerCertificate == certificate {
					continue nextInChain
				}
			}
			list[signature] = certificate
		}
	}

	for _, certificate := range list {
		certificates += string(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certificate.Raw,
		}))
	}
	return
}

func (n *client) do(uri, method string, request io.Reader, requestType string, headers http.Header) (res *http.Response, err error) {
	url, err := n.url.Parse(uri)
	if err != nil {
		return
	}

	req, err := http.NewRequest(method, url.String(), request)
	if err != nil {
		err = fmt.Errorf("failed to create NewRequest: %v", err)
		return
	}

	if headers != nil {
		req.Header = headers
	}

	if request != nil {
		req.Header.Set("Content-Type", requestType)
	}

	n.ensureTLSConfig()

	res, err = n.Do(req)
	if err != nil {
		err = fmt.Errorf("couldn't execute %v against %s: %v", req.Method, req.URL, err)
		return
	}
	return
}

func (n *client) doJSON(uri, method string, statusCode int, request interface{}, response interface{}) (int, string, string) {
	var body io.Reader

	if request != nil {
		requestBody, err := json.Marshal(request)
		if err != nil {
			return -1, fmt.Sprintf("failed to marshal project object: %v", err), ""
		}
		body = bytes.NewReader(requestBody)
	}

	headers := make(http.Header)
	if response != nil {
		headers.Set("Accept", "application/json")
	}

	res, err := n.do(uri, method, body, "application/json", headers)
	if err != nil {
		return -1, err.Error(), ""
	}
	defer res.Body.Close()
	defer io.Copy(ioutil.Discard, res.Body)

	if res.StatusCode == statusCode {
		if response != nil {
			if contentType := res.Header.Get("Content-Type"); contentType != "application/json" {
				return -1, fmt.Sprintf("Server should return application/json. Got: %v", contentType), ""
			}

			d := json.NewDecoder(res.Body)
			err = d.Decode(response)
			if err != nil {
				return -1, fmt.Sprintf("Error decoding json payload %v", err), ""
			}
		}
	}

	return res.StatusCode, res.Status, n.getCAChain(res.TLS)
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
		url:    url,
		caFile: config.TLSCAFile,
	}

	if CertificateDirectory != "" && c.caFile == "" {
		hostAndPort := strings.Split(url.Host, ":")
		c.caFile = filepath.Join(CertificateDirectory, hostAndPort[0]+".crt")
	}

	return
}
