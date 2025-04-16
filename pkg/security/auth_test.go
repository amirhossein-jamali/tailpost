package security

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/amirhossein-jamali/tailpost/pkg/config"
	"golang.org/x/oauth2"
)

func TestBasicAuthProvider(t *testing.T) {
	// Create a basic auth provider
	provider := NewBasicAuthProvider("user", "pass")

	// Test AddAuthentication
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	err := provider.AddAuthentication(req)
	if err != nil {
		t.Errorf("Unexpected error in AddAuthentication: %v", err)
	}

	// Check authorization header
	authHeader := req.Header.Get("Authorization")
	expected := "Basic dXNlcjpwYXNz" // Base64 of "user:pass"
	if authHeader != expected {
		t.Errorf("Expected Basic auth header to be '%s', got '%s'", expected, authHeader)
	}

	// Test Authenticate with valid credentials
	authenticated, err := provider.Authenticate(req)
	if err != nil {
		t.Errorf("Unexpected error in Authenticate: %v", err)
	}
	if !authenticated {
		t.Errorf("Expected authentication to succeed with valid credentials")
	}

	// Test Authenticate with invalid header
	req.Header.Set("Authorization", "Basic invalid")
	authenticated, err = provider.Authenticate(req)
	if err == nil {
		t.Errorf("Expected error with invalid base64 in auth header")
	}
	if authenticated {
		t.Errorf("Expected authentication to fail with invalid base64")
	}

	// Test Authenticate with wrong credentials
	req.Header.Set("Authorization", "Basic b3RoZXI6cGFzcw==") // "other:pass"
	authenticated, err = provider.Authenticate(req)
	if err != nil {
		t.Errorf("Unexpected error in Authenticate: %v", err)
	}
	if authenticated {
		t.Errorf("Expected authentication to fail with wrong username")
	}

	// Test Authenticate with no header
	req.Header.Del("Authorization")
	authenticated, err = provider.Authenticate(req)
	if err != nil {
		t.Errorf("Unexpected error in Authenticate: %v", err)
	}
	if authenticated {
		t.Errorf("Expected authentication to fail with no header")
	}
}

func TestTokenAuthProvider(t *testing.T) {
	// Create a temporary token file
	tokenFile, err := os.CreateTemp("", "token-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tokenFile.Name())

	// Write test token
	testToken := "test-token-12345"
	if _, err := tokenFile.Write([]byte(testToken)); err != nil {
		t.Fatalf("Failed to write to token file: %v", err)
	}
	if err := tokenFile.Close(); err != nil {
		t.Fatalf("Failed to close token file: %v", err)
	}

	// Create a token auth provider
	provider, err := NewTokenAuthProvider(tokenFile.Name())
	if err != nil {
		t.Fatalf("Failed to create token auth provider: %v", err)
	}

	// Test AddAuthentication
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	err = provider.AddAuthentication(req)
	if err != nil {
		t.Errorf("Unexpected error in AddAuthentication: %v", err)
	}

	// Check authorization header
	authHeader := req.Header.Get("Authorization")
	expected := "Bearer " + testToken
	if authHeader != expected {
		t.Errorf("Expected Bearer auth header to be '%s', got '%s'", expected, authHeader)
	}

	// Test Authenticate with valid token
	authenticated, err := provider.Authenticate(req)
	if err != nil {
		t.Errorf("Unexpected error in Authenticate: %v", err)
	}
	if !authenticated {
		t.Errorf("Expected authentication to succeed with valid token")
	}

	// Test Authenticate with wrong token
	req.Header.Set("Authorization", "Bearer wrong-token")
	authenticated, err = provider.Authenticate(req)
	if err != nil {
		t.Errorf("Unexpected error in Authenticate: %v", err)
	}
	if authenticated {
		t.Errorf("Expected authentication to fail with wrong token")
	}

	// Test error with non-existent token file
	_, err = NewTokenAuthProvider("/non/existent/file.txt")
	if err == nil {
		t.Errorf("Expected error when token file doesn't exist")
	}
}

func TestHeaderAuthProvider(t *testing.T) {
	// Create a header auth provider
	headers := map[string]string{
		"X-API-Key":   "api-key-12345",
		"X-Tenant-ID": "tenant-abc",
	}
	provider := NewHeaderAuthProvider(headers)

	// Test AddAuthentication
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	err := provider.AddAuthentication(req)
	if err != nil {
		t.Errorf("Unexpected error in AddAuthentication: %v", err)
	}

	// Check headers were set
	for k, v := range headers {
		if req.Header.Get(k) != v {
			t.Errorf("Expected header %s to be '%s', got '%s'", k, v, req.Header.Get(k))
		}
	}

	// Test Authenticate with valid headers
	authenticated, err := provider.Authenticate(req)
	if err != nil {
		t.Errorf("Unexpected error in Authenticate: %v", err)
	}
	if !authenticated {
		t.Errorf("Expected authentication to succeed with valid headers")
	}

	// Test Authenticate with missing header
	req.Header.Del("X-Tenant-ID")
	authenticated, err = provider.Authenticate(req)
	if err != nil {
		t.Errorf("Unexpected error in Authenticate: %v", err)
	}
	if authenticated {
		t.Errorf("Expected authentication to fail with missing header")
	}

	// Test Authenticate with wrong header value
	req.Header.Set("X-API-Key", "wrong-key")
	req.Header.Set("X-Tenant-ID", "tenant-abc")
	authenticated, err = provider.Authenticate(req)
	if err != nil {
		t.Errorf("Unexpected error in Authenticate: %v", err)
	}
	if authenticated {
		t.Errorf("Expected authentication to fail with wrong header value")
	}
}

// MockTokenSource is a mock implementation of oauth2.TokenSource for testing
type MockTokenSource struct {
	token *oauth2.Token
	err   error
}

func (m *MockTokenSource) Token() (*oauth2.Token, error) {
	return m.token, m.err
}

func TestOAuth2Provider(t *testing.T) {
	// Create an OAuth2 provider
	provider := NewOAuth2Provider("client-id", "client-secret", "https://example.com/token", []string{"scope1"})

	// Replace the token source with our mock
	mockToken := &oauth2.Token{
		AccessToken: "test-oauth-token",
		TokenType:   "Bearer",
	}
	provider.tokenSource = &MockTokenSource{token: mockToken}

	// Test AddAuthentication with successful token
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	err := provider.AddAuthentication(req)
	if err != nil {
		t.Errorf("Unexpected error in AddAuthentication: %v", err)
	}

	// Check the authorization header was set correctly
	authHeader := req.Header.Get("Authorization")
	expected := "Bearer test-oauth-token"
	if authHeader != expected {
		t.Errorf("Expected OAuth2 auth header to be '%s', got '%s'", expected, authHeader)
	}

	// Test with an error from the token source
	provider.tokenSource = &MockTokenSource{err: context.Canceled}
	req, _ = http.NewRequest("GET", "http://example.com", nil)
	err = provider.AddAuthentication(req)
	if err == nil {
		t.Errorf("Expected error from token source but got nil")
	}

	// Test Authenticate method which is not implemented
	authenticated, err := provider.Authenticate(req)
	if authenticated {
		t.Errorf("OAuth2 Authenticate should not return true")
	}
	if err == nil {
		t.Errorf("OAuth2 Authenticate should return an error")
	}
}

func TestNewAuthProvider(t *testing.T) {
	// Test none type
	authConfig := config.AuthConfig{
		Type: "none",
	}
	provider, err := NewAuthProvider(authConfig)
	if err != nil {
		t.Errorf("Unexpected error with none auth type: %v", err)
	}
	if provider != nil {
		t.Errorf("Expected nil provider with none auth type, got non-nil")
	}

	// Test basic auth type
	authConfig = config.AuthConfig{
		Type:     "basic",
		Username: "user",
		Password: "pass",
	}
	provider, err = NewAuthProvider(authConfig)
	if err != nil {
		t.Errorf("Unexpected error with basic auth type: %v", err)
	}
	if provider == nil {
		t.Fatalf("Expected non-nil provider with basic auth type, got nil")
	}
	_, ok := provider.(*BasicAuthProvider)
	if !ok {
		t.Errorf("Expected *BasicAuthProvider with basic auth type")
	}

	// Test header auth type
	authConfig = config.AuthConfig{
		Type: "header",
		Headers: map[string]string{
			"X-API-Key": "api-key",
		},
	}
	provider, err = NewAuthProvider(authConfig)
	if err != nil {
		t.Errorf("Unexpected error with header auth type: %v", err)
	}
	if provider == nil {
		t.Fatalf("Expected non-nil provider with header auth type, got nil")
	}
	_, ok = provider.(*HeaderAuthProvider)
	if !ok {
		t.Errorf("Expected *HeaderAuthProvider with header auth type")
	}

	// Test OAuth2 auth type
	authConfig = config.AuthConfig{
		Type:         "oauth2",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		TokenURL:     "https://example.com/token",
		Scopes:       []string{"scope1"},
	}
	provider, err = NewAuthProvider(authConfig)
	if err != nil {
		t.Errorf("Unexpected error with oauth2 auth type: %v", err)
	}
	if provider == nil {
		t.Fatalf("Expected non-nil provider with oauth2 auth type, got nil")
	}
	_, ok = provider.(*OAuth2Provider)
	if !ok {
		t.Errorf("Expected *OAuth2Provider with oauth2 auth type")
	}

	// Test token auth type (will fail because we don't have a token file)
	authConfig = config.AuthConfig{
		Type:      "token",
		TokenFile: "/non/existent/file.txt",
	}
	_, err = NewAuthProvider(authConfig)
	if err == nil {
		t.Errorf("Expected error with non-existent token file, got nil")
	}

	// Test unsupported auth type
	authConfig = config.AuthConfig{
		Type: "unsupported",
	}
	_, err = NewAuthProvider(authConfig)
	if err == nil {
		t.Errorf("Expected error with unsupported auth type, got nil")
	}
}
