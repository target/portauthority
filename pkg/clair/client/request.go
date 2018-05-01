package clairclient

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// Request struct init
type Request struct {
	Method   string
	URL      *url.URL
	Params   url.Values
	Body     io.Reader
	BodySize int64
}

// JSONBody sets the request's body to the json encoded value
func (r *Request) JSONBody(val interface{}) error {
	var buff *bytes.Buffer
	var enc *json.Encoder
	var err error

	buff = bytes.NewBuffer(nil)
	enc = json.NewEncoder(buff)

	err = enc.Encode(val)
	if err != nil {
		return errors.Wrap(err, "json encoding failed")
	}

	r.Body = buff
	r.BodySize = int64(buff.Len())
	return nil
}

// RawBody func init
func (r *Request) RawBody(raw []byte) error {
	buff := bytes.NewBuffer(raw)

	r.Body = buff
	r.BodySize = int64(buff.Len())

	return nil
}

// HTTPReq func init
func (r *Request) HTTPReq() (*http.Request, error) {
	var req *http.Request
	r.URL.RawQuery = r.Params.Encode()

	req, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		return nil, errors.Wrap(err, "creating http request failed")
	}
	return req, nil
}
