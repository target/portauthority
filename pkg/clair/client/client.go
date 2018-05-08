// Copyright (c) 2018 Target Brands, Inc.

package clairclient

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
)

// Config for Clair Client
type Config struct {
	Address    string
	HTTPClient *http.Client
}

// TLSConfig for Clair Client
type TLSConfig struct {
	CaCert   string
	Insecure bool
}

// DefaultConfig will return a client configuration with default values
func DefaultConfig() *Config {
	var config *Config

	transport := &http.Transport{
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	config = &Config{
		Address: "http://127.0.0.1:6060",
		HTTPClient: &http.Client{
			Transport: transport,
			Timeout:   time.Second * 10,
		},
	}

	return config
}

// ConfigureTLS will apply the provided tlsconfig to the config
func (c *Config) ConfigureTLS(tc *TLSConfig) error {
	clientTLSConfig := c.HTTPClient.Transport.(*http.Transport).TLSClientConfig

	if tc.CaCert != "" {
		caCert, err := ioutil.ReadFile(tc.CaCert)
		if err != nil {
			return errors.Wrap(err, "failed to read cacert file")
		}
		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM(caCert)
		clientTLSConfig.RootCAs = certPool
	}

	clientTLSConfig.InsecureSkipVerify = tc.Insecure

	return nil
}

// Client struct
type Client struct {
	addr   *url.URL
	config *Config
}

// NewClient Sets up a new Clair client
func NewClient(c *Config) (*Client, error) {
	if nil == c {
		c = DefaultConfig()
	}

	u, err := url.Parse(c.Address)
	if err != nil {
		return nil, errors.Wrap(err, "parsing url failed")
	}

	if c.HTTPClient == nil {
		c.HTTPClient = DefaultConfig().HTTPClient
	}

	client := &Client{
		addr:   u,
		config: c,
	}

	return client, nil
}

// Request builds the standard request to Clair
func (c *Client) Request(r *Request) (*http.Response, error) {
	var req *http.Request
	var result *http.Response
	var err error

	req, err = r.HTTPReq()
	if err != nil {
		return nil, errors.Wrap(err, "creating http request from logical request failed")
	}

	result, err = c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "doing http request failed")
	}

	return result, nil
}

// DecodeJSONBody decodes the body from a response into the provided interface
func DecodeJSONBody(resp *http.Response, out interface{}) error {
	var dec *json.Decoder

	if nil == resp.Body {
		return errors.New("Response body is nil")
	}

	if nil == out {
		return errors.New("Output interface is nil")
	}

	dec = json.NewDecoder(resp.Body)

	return dec.Decode(out)
}
