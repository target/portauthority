package registry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// TokenTransport struct init
type TokenTransport struct {
	Transport http.RoundTripper
	Username  string
	Password  string
}

// RoundTrip func init
func (t *TokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if AuthService := isTokenDemand(resp); AuthService != nil {
		if resp != nil {
			resp.Body.Close()
		}
		resp, err = t.authAndRetry(AuthService, req)
	}
	return resp, err
}

type authToken struct {
	Token string `json:"token"`
}

func (t *TokenTransport) authAndRetry(AuthService *AuthService, req *http.Request) (*http.Response, error) {
	token, authResp, err := t.auth(AuthService)
	if err != nil {
		return authResp, err
	}

	retryResp, err := t.retry(req, token)
	return retryResp, err
}

func (t *TokenTransport) auth(AuthService *AuthService) (string, *http.Response, error) {
	authReq, err := AuthService.Request(t.Username, t.Password)
	if err != nil {
		return "", nil, err
	}

	client := http.Client{
		Transport: t.Transport,
	}

	response, err := client.Do(authReq)
	if err != nil {
		return "", nil, err
	}

	if response.StatusCode != http.StatusOK {
		return "", response, err
	}
	defer response.Body.Close()

	var myAuthToken authToken
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&myAuthToken)
	if err != nil {
		return "", nil, err
	}

	return myAuthToken.Token, nil, nil
}

func (t *TokenTransport) retry(req *http.Request, token string) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := t.Transport.RoundTrip(req)
	return resp, err
}

// AuthService struct init
type AuthService struct {
	Realm   string
	Service string
	Scope   string
}

// Request func init
func (AuthService *AuthService) Request(username, password string) (*http.Request, error) {
	url, err := url.Parse(AuthService.Realm)
	if err != nil {
		return nil, err
	}

	q := url.Query()
	q.Set("service", AuthService.Service)
	if AuthService.Scope != "" {
		q.Set("scope", AuthService.Scope)
	}
	url.RawQuery = q.Encode()

	request, err := http.NewRequest("GET", url.String(), nil)

	if username != "" || password != "" {
		request.SetBasicAuth(username, password)
	}

	return request, err
}

func isTokenDemand(resp *http.Response) *AuthService {
	if resp == nil {
		return nil
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return nil
	}
	return ParseOauthHeader(resp)
}

// ParseOauthHeader func init
func ParseOauthHeader(resp *http.Response) *AuthService {
	challenges := ParseAuthHeader(resp.Header)
	for _, challenge := range challenges {
		if challenge.Scheme == "bearer" {
			return &AuthService{
				Realm:   challenge.Parameters["realm"],
				Service: challenge.Parameters["service"],
				Scope:   challenge.Parameters["scope"],
			}
		}
	}
	return nil
}
