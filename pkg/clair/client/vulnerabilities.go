package clairclient

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// Vulnerability struct init
type Vulnerability struct {
	Name          string                 `json:"Name,omitempty"`
	NamespaceName string                 `json:"NamespaceName,omitempty"`
	Description   string                 `json:"Description,omitempty"`
	Link          string                 `json:"Link,omitempty"`
	Severity      string                 `json:"Severity,omitempty"`
	Metadata      map[string]interface{} `json:"Metadata,omitempty"`
	FixedBy       string                 `json:"FixedBy,omitempty"`
	FixedIn       []Feature              `json:"FixedIn,omitempty"`
}

// VulnerabilityWithLayers struct init
type VulnerabilityWithLayers struct {
	Vulnerability *Vulnerability `json:"Vulnerability,omitempty"`

	// This field is guaranteed to be in order only for pagination.
	// Indices from different notifications may not be comparable.
	OrderedLayersIntroducingVulnerability []OrderedLayerName `json:"OrderedLayersIntroducingVulnerability,omitempty"`

	// This field is deprecated.
	LayersIntroducingVulnerability []string `json:"LayersIntroducingVulnerability,omitempty"`
}

// VulnerabilityEnvelope struct init
type VulnerabilityEnvelope struct {
	Vulnerability   *Vulnerability   `json:"Vulnerability,omitempty"`
	Vulnerabilities *[]Vulnerability `json:"Vulnerabilities,omitempty"`
	NextPage        string           `json:"NextPage,omitempty"`
	Error           *Error           `json:"Error,omitempty"`
}

// GetVulnerabilities init
func (c *Client) GetVulnerabilities(nspace string) (*VulnerabilityEnvelope, error) {
	reqURL := &url.URL{
		Scheme: c.addr.Scheme,
		Host:   c.addr.Host,
		Path:   fmt.Sprintf("/v1/namespaces/%s/vulnerabilities", nspace),
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

	ve := &VulnerabilityEnvelope{}
	err = DecodeJSONBody(resp, ve)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing response")
	}

	return ve, nil
}

// PostVulnerabilities init
func (c *Client) PostVulnerabilities(vuln *Vulnerability) (*VulnerabilityEnvelope, error) {
	reqURL := &url.URL{
		Scheme: c.addr.Scheme,
		Host:   c.addr.Host,
		Path:   fmt.Sprintf("/v1/namespaces/%s/vulnerabilities", vuln.NamespaceName),
	}

	req := &Request{
		Method: "POST",
		URL:    reqURL,
		Params: make(map[string][]string),
	}

	err := req.JSONBody(&VulnerabilityEnvelope{Vulnerability: vuln})
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling json body")
	}

	resp, err := c.Request(req)
	if err != nil {
		return nil, errors.Wrap(err, "error performing request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, newStatusCodeError(resp.StatusCode)
	}

	respVE := &VulnerabilityEnvelope{}
	err = DecodeJSONBody(resp, respVE)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing response")
	}

	return respVE, nil
}

// GetVulnerability init
func (c *Client) GetVulnerability(nspace, vuln string) (*VulnerabilityEnvelope, error) {
	reqURL := &url.URL{
		Scheme: c.addr.Scheme,
		Host:   c.addr.Host,
		Path:   fmt.Sprintf("/v1/namespaces/%s/vulnerabilities/%s", nspace, vuln),
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

	ve := &VulnerabilityEnvelope{}
	err = DecodeJSONBody(resp, ve)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing response")
	}

	return ve, nil
}

// PutVulnerbaility init
func (c *Client) PutVulnerbaility(vuln *Vulnerability) (*VulnerabilityEnvelope, error) {
	reqURL := &url.URL{
		Scheme: c.addr.Scheme,
		Host:   c.addr.Host,
		Path:   fmt.Sprintf("/v1/namespaces/%s/vulnerabilities/%s", vuln.NamespaceName, vuln.Name),
	}

	req := &Request{
		Method: "PUT",
		URL:    reqURL,
		Params: make(map[string][]string),
	}

	err := req.JSONBody(vuln)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling json bodyy")
	}

	resp, err := c.Request(req)
	if err != nil {
		return nil, errors.Wrap(err, "error performing request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, newStatusCodeError(resp.StatusCode)
	}

	ve := &VulnerabilityEnvelope{}
	err = DecodeJSONBody(resp, ve)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing response")
	}

	return ve, nil
}

// DeleteVulnerbaility func init
func (c *Client) DeleteVulnerbaility(nspace, vuln string) error {
	reqURL := &url.URL{
		Scheme: c.addr.Scheme,
		Host:   c.addr.Host,
		Path:   fmt.Sprintf("/v1/namespaces/%s/vulnerabilities/%s", nspace, vuln),
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
