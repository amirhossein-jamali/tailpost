package security

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/amirhossein-jamali/tailpost/pkg/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// AuthProvider is an interface for authentication providers
type AuthProvider interface {
	// AddAuthentication adds authentication to the given request
	AddAuthentication(req *http.Request) error
	// Authenticate authenticates a request (for server-side)
	Authenticate(req *http.Request) (bool, error)
}

// BasicAuthProvider implements basic authentication
type BasicAuthProvider struct {
	Username string
	Password string
}

// NewBasicAuthProvider creates a new basic auth provider
func NewBasicAuthProvider(username, password string) *BasicAuthProvider {
	return &BasicAuthProvider{
		Username: username,
		Password: password,
	}
}

// AddAuthentication adds basic auth to the request
func (p *BasicAuthProvider) AddAuthentication(req *http.Request) error {
	auth := p.Username + ":" + p.Password
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Set("Authorization", "Basic "+encoded)
	return nil
}

// Authenticate checks basic auth credentials
func (p *BasicAuthProvider) Authenticate(req *http.Request) (bool, error) {
	auth := req.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Basic ") {
		return false, nil
	}

	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
	if err != nil {
		return false, fmt.Errorf("error decoding auth header: %v", err)
	}

	pair := strings.SplitN(string(payload), ":", 2)
	if len(pair) != 2 {
		return false, nil
	}

	return pair[0] == p.Username && pair[1] == p.Password, nil
}

// TokenAuthProvider implements token-based authentication
type TokenAuthProvider struct {
	Token string
}

// NewTokenAuthProvider creates a new token auth provider
func NewTokenAuthProvider(tokenFile string) (*TokenAuthProvider, error) {
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("error reading token file: %v", err)
	}

	token := strings.TrimSpace(string(data))
	return &TokenAuthProvider{Token: token}, nil
}

// AddAuthentication adds token auth to the request
func (p *TokenAuthProvider) AddAuthentication(req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+p.Token)
	return nil
}

// Authenticate validates the token
func (p *TokenAuthProvider) Authenticate(req *http.Request) (bool, error) {
	auth := req.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return false, nil
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	return token == p.Token, nil
}

// OAuth2Provider implements OAuth2 authentication
type OAuth2Provider struct {
	tokenSource oauth2.TokenSource
}

// NewOAuth2Provider creates a new OAuth2 provider
func NewOAuth2Provider(clientID, clientSecret, tokenURL string, scopes []string) *OAuth2Provider {
	config := clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
		Scopes:       scopes,
	}

	return &OAuth2Provider{
		tokenSource: config.TokenSource(context.Background()),
	}
}

// AddAuthentication adds OAuth2 token to the request
func (p *OAuth2Provider) AddAuthentication(req *http.Request) error {
	token, err := p.tokenSource.Token()
	if err != nil {
		return fmt.Errorf("error getting OAuth2 token: %v", err)
	}

	token.SetAuthHeader(req)
	return nil
}

// Authenticate for OAuth2 is not implemented for server-side in this example
func (p *OAuth2Provider) Authenticate(req *http.Request) (bool, error) {
	return false, fmt.Errorf("OAuth2 server-side authentication not implemented")
}

// HeaderAuthProvider implements custom header authentication
type HeaderAuthProvider struct {
	Headers map[string]string
}

// NewHeaderAuthProvider creates a new header auth provider
func NewHeaderAuthProvider(headers map[string]string) *HeaderAuthProvider {
	return &HeaderAuthProvider{
		Headers: headers,
	}
}

// AddAuthentication adds custom headers for authentication
func (p *HeaderAuthProvider) AddAuthentication(req *http.Request) error {
	for key, value := range p.Headers {
		req.Header.Set(key, value)
	}
	return nil
}

// Authenticate checks if request has the expected header values
func (p *HeaderAuthProvider) Authenticate(req *http.Request) (bool, error) {
	for key, expectedValue := range p.Headers {
		actualValue := req.Header.Get(key)
		if actualValue != expectedValue {
			return false, nil
		}
	}
	return true, nil
}

// NewAuthProvider creates an authentication provider based on configuration
func NewAuthProvider(authConfig config.AuthConfig) (AuthProvider, error) {
	switch authConfig.Type {
	case "none":
		return nil, nil
	case "basic":
		return NewBasicAuthProvider(authConfig.Username, authConfig.Password), nil
	case "token":
		return NewTokenAuthProvider(authConfig.TokenFile)
	case "oauth2":
		return NewOAuth2Provider(
			authConfig.ClientID,
			authConfig.ClientSecret,
			authConfig.TokenURL,
			authConfig.Scopes,
		), nil
	case "header":
		return NewHeaderAuthProvider(authConfig.Headers), nil
	default:
		return nil, fmt.Errorf("unsupported authentication type: %s", authConfig.Type)
	}
}
