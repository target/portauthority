package clairclient

import (
	"net/http"
	"net/url"

	"fmt"

	"strconv"

	"github.com/pkg/errors"
)

// Notification struct init
type Notification struct {
	Name     string                   `json:"Name,omitempty"`
	Created  string                   `json:"Created,omitempty"`
	Notified string                   `json:"Notified,omitempty"`
	Deleted  string                   `json:"Deleted,omitempty"`
	Limit    int                      `json:"Limit,omitempty"`
	Page     string                   `json:"Page,omitempty"`
	NextPage string                   `json:"NextPage,omitempty"`
	Old      *VulnerabilityWithLayers `json:"Old,omitempty"`
	New      *VulnerabilityWithLayers `json:"New,omitempty"`
}

// NotificationEnvelope struct init
type NotificationEnvelope struct {
	Notification *Notification `json:"Notification,omitempty"`
	Error        *Error        `json:"Error,omitempty"`
}

// GetNotifications func init
func (c *Client) GetNotifications(name, page string, limit int) (*NotificationEnvelope, error) {

	params := make(map[string][]string)
	if page != "" {
		params["page"] = []string{page}
	}

	if limit != 0 {
		params["limit"] = []string{strconv.Itoa(limit)}
	}
	reqURL := &url.URL{
		Scheme: c.addr.Scheme,
		Host:   c.addr.Host,
		Path:   fmt.Sprintf("/v1/notifications/%s", name),
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

	ne := &NotificationEnvelope{}
	err = DecodeJSONBody(resp, ne)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing response")
	}

	return ne, nil
}

// DeleteNotifications func init
func (c *Client) DeleteNotifications(name string) error {
	reqURL := &url.URL{
		Scheme: c.addr.Scheme,
		Host:   c.addr.Host,
		Path:   fmt.Sprintf("/v1/notifications/%s", name),
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
