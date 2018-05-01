package clairclient

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// GetFixes func init
func (c *Client) GetFixes(nspace, vuln string) (*FeatureEnvelope, error) {
	reqURL := &url.URL{
		Scheme: c.addr.Scheme,
		Host:   c.addr.Host,
		Path:   fmt.Sprintf("/v1/namespaces/%s/vulnerabilities/%s/fixes", nspace, vuln),
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

	fe := &FeatureEnvelope{}
	err = DecodeJSONBody(resp, fe)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing response")
	}

	return fe, nil
}

// PutFixes func init
func (c *Client) PutFixes(vuln string, feat *Feature) (*FeatureEnvelope, error) {
	reqURL := &url.URL{
		Scheme: c.addr.Scheme,
		Host:   c.addr.Host,
		Path:   fmt.Sprintf("/v1/namespaces/%s/vulnerabilities/%s/fixes/%s", feat.NamespaceName, vuln, feat.Name),
	}

	req := &Request{
		Method: "PUT",
		URL:    reqURL,
		Params: make(map[string][]string),
	}

	err := req.JSONBody(&FeatureEnvelope{Feature: feat})
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling json body")
	}

	resp, err := c.Request(req)
	if err != nil {
		return nil, errors.Wrap(err, "error performing request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, newStatusCodeError(resp.StatusCode)
	}

	fe := &FeatureEnvelope{}
	err = DecodeJSONBody(resp, fe)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing response")
	}

	return fe, nil
}

// DeleteFixes func init
func (c *Client) DeleteFixes(nspace, vuln, feat string) error {
	reqURL := &url.URL{
		Scheme: c.addr.Scheme,
		Host:   c.addr.Host,
		Path:   fmt.Sprintf("/v1/namespaces/%s/vulnerabilities/%s/fixes/%s", nspace, vuln, feat),
	}

	req := &Request{
		Method: "DELETE",
		URL:    reqURL,
		Params: make(map[string][]string),
	}

	resp, err := c.Request(req)
	if err != nil {
		return errors.Wrap(err, "error performing request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return newStatusCodeError(resp.StatusCode)
	}

	return nil
}
