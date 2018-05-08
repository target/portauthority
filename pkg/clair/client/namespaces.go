// Copyright (c) 2018 Target Brands, Inc.

package clairclient

import (
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// Namespace struct init
type Namespace struct {
	Name          string `json:"Name,omitempty"`
	VersionFormat string `json:"VersionFormat,omitempty"`
}

// NamespaceEnvelope struct init
type NamespaceEnvelope struct {
	Namespaces *[]Namespace `json:"Namespaces,omitempty"`
	Error      *Error       `json:"Error,omitempty"`
}

// GetNamespaces func init
func (c *Client) GetNamespaces() (*NamespaceEnvelope, error) {
	reqURL := &url.URL{
		Scheme: c.addr.Scheme,
		Host:   c.addr.Host,
		Path:   "/v1/namespaces",
	}

	req := &Request{
		Method: "GET",
		URL:    reqURL,
		Params: make(map[string][]string),
	}

	resp, err := c.Request(req)
	if err != nil {
		return nil, errors.Wrap(err, "error performing request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, newStatusCodeError(resp.StatusCode)
	}

	ne := &NamespaceEnvelope{}
	err = DecodeJSONBody(resp, ne)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing response")
	}

	return ne, nil
}
