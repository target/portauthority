package registry

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

// HTTPStatusError struct init
type HTTPStatusError struct {
	Response *http.Response

	// Copied from `Response.Body` to avoid problems with unclosed bodies later.
	// Nobody calls `err.Response.Body.Close()`, ever.
	Body []byte
}

func (err *HTTPStatusError) Error() string {
	return fmt.Sprintf("http: non-successful response (status=%v body=%q)", err.Response.StatusCode, err.Body)
}

var _ error = &HTTPStatusError{}

// ErrorTransport struct init
type ErrorTransport struct {
	Transport http.RoundTripper
}

// RoundTrip func init
func (t *ErrorTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	resp, err := t.Transport.RoundTrip(request)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("http: empty response")
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("http: failed to read response body (status=%v, err=%q)", resp.StatusCode, err)
		}

		return nil, &HTTPStatusError{
			Response: resp,
			Body:     body,
		}
	}

	return resp, err
}
