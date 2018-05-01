package clairclient

import (
	"net/http"
	"net/url"

	"fmt"

	"github.com/pkg/errors"
)

// Error struct init
type Error struct {
	Message string `json:"Message,omitempty"`
}

// Layer struct init
type Layer struct {
	Name             string            `json:"Name,omitempty"`
	NamespaceName    string            `json:"NamespaceName,omitempty"`
	Path             string            `json:"Path,omitempty"`
	Headers          map[string]string `json:"Headers,omitempty"`
	ParentName       string            `json:"ParentName,omitempty"`
	Format           string            `json:"Format,omitempty"`
	IndexedByVersion int               `json:"IndexedByVersion,omitempty"`
	Features         []Feature         `json:"Features,omitempty"`
}

// Feature struct init
type Feature struct {
	Name            string          `json:"Name,omitempty"`
	NamespaceName   string          `json:"NamespaceName,omitempty"`
	VersionFormat   string          `json:"VersionFormat,omitempty"`
	Version         string          `json:"Version,omitempty"`
	Vulnerabilities []Vulnerability `json:"Vulnerabilities,omitempty"`
	AddedBy         string          `json:"AddedBy,omitempty"`
}

// OrderedLayerName struct init
type OrderedLayerName struct {
	Index     int    `json:"Index"`
	LayerName string `json:"LayerName"`
}

// LayerEnvelope struct init
type LayerEnvelope struct {
	Layer *Layer `json:"Layer,omitempty"`
	Error *Error `json:"Error,omitempty"`
}

// FeatureEnvelope struct init
type FeatureEnvelope struct {
	Feature  *Feature   `json:"Feature,omitempty"`
	Features *[]Feature `json:"Features,omitempty"`
	Error    *Error     `json:"Error,omitempty"`
}

// PostLayers func init
func (c *Client) PostLayers(layer *Layer) (*LayerEnvelope, error) {

	reqURL := &url.URL{
		Scheme: c.addr.Scheme,
		Host:   c.addr.Host,
		Path:   "/v1/layers",
	}

	req := &Request{
		Method: "POST",
		URL:    reqURL,
		Params: make(map[string][]string),
	}

	le := &LayerEnvelope{
		Layer: layer,
	}

	err := req.JSONBody(le)
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

	respLE := &LayerEnvelope{}
	err = DecodeJSONBody(resp, respLE)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing response")
	}

	return respLE, nil
}

// GetLayers func init
func (c *Client) GetLayers(name string, withFeatures, withVulnerabilities bool) (*LayerEnvelope, error) {

	path := fmt.Sprintf("/v1/layers/%s", name)
	parms := url.Values{}

	// If vulnerabilites exits, it automatically returns features
	if withVulnerabilities {
		parms.Add("vulnerabilities", "")
	} else if withFeatures {
		parms.Add("features", "")
	}

	reqURL := &url.URL{
		Scheme: c.addr.Scheme,
		Host:   c.addr.Host,
		Path:   path,
	}

	req := &Request{
		Method: "GET",
		URL:    reqURL,
		Params: parms,
	}

	resp, err := c.Request(req)
	if err != nil {
		return nil, errors.Wrap(err, "error performing request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, newStatusCodeError(resp.StatusCode)
	}

	respLE := &LayerEnvelope{}
	err = DecodeJSONBody(resp, respLE)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing response")
	}

	return respLE, nil
}

// DeleteLayers func init
func (c *Client) DeleteLayers(name string) error {
	reqURL := &url.URL{
		Scheme: c.addr.Scheme,
		Host:   c.addr.Host,
		Path:   fmt.Sprintf("/v1/layers/%s", name),
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
