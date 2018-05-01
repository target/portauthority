package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/target/portauthority/pkg/docker/registry"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/google"
)

// AuthConfig struct init
type AuthConfig struct {

	// RegistryURL, Username, Password are required for obtaining a token.
	// Repo and Tag are required for docker.io as as tokens must have a directed
	// scope.
	RegistryURL string
	Repo        string
	Tag         string
	Username    string
	Password    string
}

// Token struct init
type Token struct {
	Token string `json:"token"`
}

// AuthRegistry will attempt to authenticate to a register with the provided
// credentials, returning the resulting token.
func AuthRegistry(authConfig *AuthConfig) (*Token, error) {

	token := &Token{}

	// Need special handling for obtaining the GCR token at this point in time.
	// TODO: At some point, the token should be obtained from the Docker client
	// once a valid GCR token can be extracted.
	if strings.Contains(authConfig.RegistryURL, "gcr.io") && authConfig.Password != "" { // gcr\.io$
		jwtConfig, err := google.JWTConfigFromJSON([]byte(authConfig.Password), "https://www.googleapis.com/auth/devstorage.read_only")
		if err != nil {
			return token, errors.Wrap(err, "error getting gcr token")
		}

		gcrtoken, err := jwtConfig.TokenSource(context.Background()).Token()
		if err != nil {
			return token, errors.Wrap(err, "getting gcr token error")
		}

		token = &Token{Token: gcrtoken.AccessToken}
		return token, nil
	}

	// Format appropriate URL
	url := fmt.Sprintf("%s/v2/", authConfig.RegistryURL)

	if authConfig.Repo != "" && authConfig.Tag != "" {
		url = fmt.Sprintf("%s/v2/%s/manifests/%s", authConfig.RegistryURL, authConfig.Repo, authConfig.Tag)
	}

	resp, err := http.Get(url)
	if resp == nil {
		return token, errors.Wrap(err, "no response")
	}
	if err != nil {
		return token, errors.Wrap(err, "error making initial request to registry for auth")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {

		authService := registry.ParseOauthHeader(resp)

		authReq, err := authService.Request(authConfig.Username, authConfig.Password)
		if err != nil {
			return token, errors.Wrap(err, "error building auth request")
		}

		tokenResp, err := http.DefaultClient.Do(authReq)
		if err != nil {
			return token, errors.Wrap(err, "error performing token request")
		}
		defer tokenResp.Body.Close()

		var t Token
		decoder := json.NewDecoder(tokenResp.Body)
		err = decoder.Decode(&t)
		if err != nil {
			return token, errors.Wrap(err, "error decoding token response")
		}

		token = &Token{Token: t.Token}

		return token, nil
	}

	return token, nil
}
