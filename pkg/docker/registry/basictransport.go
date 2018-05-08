// Copyright (c) 2015, Salesforce.com, Inc. All rights reserved.

package registry

import (
	"net/http"
	"strings"
)

// BasicTransport struct init
type BasicTransport struct {
	Transport http.RoundTripper
	URL       string
	Username  string
	Password  string
}

// RoundTrip func init
func (t *BasicTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(req.URL.String(), t.URL) {
		if t.Username != "" || t.Password != "" {
			req.SetBasicAuth(t.Username, t.Password)
		}
	}
	resp, err := t.Transport.RoundTrip(req)
	return resp, err
}
