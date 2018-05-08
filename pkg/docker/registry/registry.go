// Copyright (c) 2015, Salesforce.com, Inc. All rights reserved.

package registry

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
)

// LogfCallback func init
type LogfCallback func(format string, args ...interface{})

// Quiet function discards log messages silently
func Quiet(format string, args ...interface{}) {
	/* discard logs */
}

// Log func passes log messages along to Go's "log" module
func Log(format string, args ...interface{}) {
	log.Debug(fmt.Sprintf(format, args...))
}

// Registry struct init
type Registry struct {
	URL    string
	Client *http.Client
	Logf   LogfCallback
}

// New Func init
/*
 * Create a new Registry with the given URL and credentials, then Ping()s it
 * before returning it to verify that the Registry is available.
 *
 * Alternately, you can construct a Registry manually by populating the fields.
 * This passes http.DefaultTransport to WrapTransport when creating the
 * http.Client.
 */
func New(registryURL, username, password string) (*Registry, error) {
	transport := http.DefaultTransport

	return newFromTransport(registryURL, username, password, transport, Log)
}

// NewInsecure func init
/*
 * Create a new Registry, as with New, using an http.Transport that disables
 * SSL certificate verification.
 */
func NewInsecure(registryURL, username, password string) (*Registry, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return newFromTransport(registryURL, username, password, transport, Log)
}

// WrapTransport func init
/*
 * Given an existing http.RoundTripper such as http.DefaultTransport, build the
 * transport stack necessary to authenticate to the Docker registry API. This
 * adds in support for OAuth bearer tokens and HTTP Basic auth, and sets up
 * error handling this library relies on.
 */
func WrapTransport(transport http.RoundTripper, url, username, password string) http.RoundTripper {
	tokenTransport := &TokenTransport{
		Transport: transport,
		Username:  username,
		Password:  password,
	}
	basicAuthTransport := &BasicTransport{
		Transport: tokenTransport,
		URL:       url,
		Username:  username,
		Password:  password,
	}
	errorTransport := &ErrorTransport{
		Transport: basicAuthTransport,
	}
	return errorTransport
}

func newFromTransport(registryURL, username, password string, transport http.RoundTripper, logf LogfCallback) (*Registry, error) {
	url := strings.TrimSuffix(registryURL, "/")
	transport = WrapTransport(transport, url, username, password)
	registry := &Registry{
		URL: url,
		Client: &http.Client{
			Transport: transport,
		},
		Logf: logf,
	}

	if err := registry.Ping(); err != nil {
		return nil, err
	}

	return registry, nil
}

func (r *Registry) url(pathTemplate string, args ...interface{}) string {
	pathSuffix := fmt.Sprintf(pathTemplate, args...)
	url := fmt.Sprintf("%s%s", r.URL, pathSuffix)
	return url
}

// Ping func init
func (r *Registry) Ping() error {
	url := r.url("/v2/")
	r.Logf("registry.ping url=%s", url)
	resp, err := r.Client.Get(url)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}
	return nil
}
