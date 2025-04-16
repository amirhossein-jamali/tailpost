package security

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"

	"github.com/amirhossein-jamali/tailpost/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTLSConfig(t *testing.T) {
	// Test with TLS disabled
	cfg := config.TLSConfig{
		Enabled: false,
	}
	tlsConfig, err := CreateTLSConfig(cfg)
	if err != nil {
		t.Errorf("Unexpected error with TLS disabled: %v", err)
	}
	if tlsConfig != nil {
		t.Errorf("Expected nil TLS config when TLS is disabled, got non-nil")
	}

	// Test with TLS enabled but minimal settings
	cfg = config.TLSConfig{
		Enabled:            true,
		InsecureSkipVerify: true,
		ServerName:         "example.com",
	}
	tlsConfig, err = CreateTLSConfig(cfg)
	if err != nil {
		t.Errorf("Unexpected error with minimal TLS config: %v", err)
	}
	if tlsConfig == nil {
		t.Fatalf("Expected non-nil TLS config, got nil")
	}
	if !tlsConfig.InsecureSkipVerify {
		t.Errorf("Expected InsecureSkipVerify to be true")
	}
	if tlsConfig.ServerName != "example.com" {
		t.Errorf("Expected ServerName to be 'example.com', got '%s'", tlsConfig.ServerName)
	}

	// Test with TLS version settings
	cfg = config.TLSConfig{
		Enabled:            true,
		InsecureSkipVerify: false,
		MinVersion:         "tls12",
		MaxVersion:         "tls13",
	}
	tlsConfig, err = CreateTLSConfig(cfg)
	if err != nil {
		t.Errorf("Unexpected error with TLS version config: %v", err)
	}
	if tlsConfig == nil {
		t.Fatalf("Expected non-nil TLS config, got nil")
	}
	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("Expected MinVersion to be TLS 1.2, got %v", tlsConfig.MinVersion)
	}
	if tlsConfig.MaxVersion != tls.VersionTLS13 {
		t.Errorf("Expected MaxVersion to be TLS 1.3, got %v", tlsConfig.MaxVersion)
	}

	// Test with invalid TLS version
	cfg = config.TLSConfig{
		Enabled:    true,
		MinVersion: "invalid",
	}
	_, err = CreateTLSConfig(cfg)
	if err == nil {
		t.Errorf("Expected error with invalid TLS version, got nil")
	}

	// Test with invalid maximum TLS version
	cfg = config.TLSConfig{
		Enabled:    true,
		MaxVersion: "invalid",
	}
	_, err = CreateTLSConfig(cfg)
	if err == nil {
		t.Errorf("Expected error with invalid max TLS version, got nil")
	}

	// Test PreferServerCipherSuites
	cfg = config.TLSConfig{
		Enabled:                  true,
		PreferServerCipherSuites: true,
	}
	tlsConfig, err = CreateTLSConfig(cfg)
	if err != nil {
		t.Errorf("Unexpected error with PreferServerCipherSuites: %v", err)
	}
	if tlsConfig == nil {
		t.Fatalf("Expected non-nil TLS config, got nil")
	}
	if !tlsConfig.PreferServerCipherSuites {
		t.Errorf("Expected PreferServerCipherSuites to be true")
	}
}

func TestCreateTLSConfigWithCertificates(t *testing.T) {
	// Create test directory for certificates
	testDir, err := os.MkdirTemp("", "tls-test")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	// Create dummy CA cert file
	caFile := filepath.Join(testDir, "ca.crt")
	err = os.WriteFile(caFile, []byte("-----BEGIN CERTIFICATE-----\nMIIDazCCAlOgAwIBAgIUJsw1ius/WyqWrZ6LgYQ6UKTpdGUwDQYJKoZIhvcNAQEL\nBQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM\nGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yMjA1MDEwMDAwMDBaFw0yMzA1\nMDEwMDAwMDBaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw\nHwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3DQEB\nAQUAA4IBDwAwggEKAoIBAQDQB7UQpOyqVjFUTLf3DQmtLLnIVrESL8jM6o7MFt4H\nrwflqVHQOesX6eMGhfJjQJuB3fENo4IpXLnHWuKIiTdqwc1h6Dlc85i8mCzxkf6X\nApFXU+Iq/sRH5XQTJgXiwm4YYXn5LAV7uSvVkf5YVxC3iUzCuzQGwhJrWxQtBkP+\nzilLKrbr5NzL0XGkHWzZVdo5q8i17eUEYcVnEbKvYo5JLgYlm1P4KZAX6nKqMg6h\ndhOYwVp1xyFLp6h8+CKGCgjUKpWW2jku0QUjxCj5hvOQaPoZVmbYxfHVUaXTr9p5\nZJa0Y0man30UdGSgS9U4ltv8bxGqOYLz4B9EOWRbYJAtAgMBAAGjUzBRMB0GA1Ud\nDgQWBBR8UhEA4juQQwegPYD8tw8JagGU2DAfBgNVHSMEGDAWgBR8UhEA4juQQweg\nPYD8tw8JagGU2DAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQC3\n5tGHBpdZK3ww1UvVBjgM15+VvMYA2QgiHMEwgGK9pYJ4drzjzLzJ9HFADaDIkZZJ\nBSLilLf9WVWgG3XqQ9l8NU29dCUgHBQGgX5GdLAkXKVrYaGPqXrY+kQSd1XWqQkM\nK4LgPXjXM0A8j2zKB5KaJW3p4XSgPzm+9cqJCVD0Lq60G+DmQiH3mZzV8xSINDIR\nrBJn5ztxVsBFZEOLFE6XRIXg1XVEPqGHJFYUDXNZGIuGZbXmRL+EIQP2OlfUy9c8\n6RrKQlKxknPBgZPBJKjiQKhClONKE7jAFCK+mJMCJGZ6jbwULUhT8YQ9xGXdT4bZ\nLcQYd8Oh0E2zW6TD5BO4\n-----END CERTIFICATE-----\n"), 0600)
	require.NoError(t, err)

	// Create dummy certificate and key files
	certFile := filepath.Join(testDir, "server.crt")
	keyFile := filepath.Join(testDir, "server.key")

	err = os.WriteFile(certFile, []byte("-----BEGIN CERTIFICATE-----\nMIIDazCCAlOgAwIBAgIUJsw1ius/WyqWrZ6LgYQ6UKTpdGUwDQYJKoZIhvcNAQEL\nBQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM\nGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yMjA1MDEwMDAwMDBaFw0yMzA1\nMDEwMDAwMDBaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw\nHwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3DQEB\nAQUAA4IBDwAwggEKAoIBAQDQB7UQpOyqVjFUTLf3DQmtLLnIVrESL8jM6o7MFt4H\nrwflqVHQOesX6eMGhfJjQJuB3fENo4IpXLnHWuKIiTdqwc1h6Dlc85i8mCzxkf6X\nApFXU+Iq/sRH5XQTJgXiwm4YYXn5LAV7uSvVkf5YVxC3iUzCuzQGwhJrWxQtBkP+\nzilLKrbr5NzL0XGkHWzZVdo5q8i17eUEYcVnEbKvYo5JLgYlm1P4KZAX6nKqMg6h\ndhOYwVp1xyFLp6h8+CKGCgjUKpWW2jku0QUjxCj5hvOQaPoZVmb"), 0600)
	require.NoError(t, err)

	err = os.WriteFile(keyFile, []byte("-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDQB7UQpOyqVjFU\nTLf3DQmtLLnIVrESL8jM6o7MFt4HrwflqVHQOesX6eMGhfJjQJuB3fENo4IpXLnH\nWuKIiTdqwc1h6Dlc85i8mCzxkf6XApFXU+Iq/sRH5XQTJgXiwm4YYXn5LAV7uSvV\nkf5YVxC3iUzCuzQGwhJrWxQtBkP+zilLKrbr5NzL0XGkHWzZVdo5q8i17eUEYcVn\nEbKvYo5JLgYlm1P4KZAX6nKqMg6hdhOYwVp1xyFLp6h8+CKGCgjUKpWW2jku0QUj\nxCj5hvOQaPoZVmb"), 0600)
	require.NoError(t, err)

	// Test with non-existent CA file
	cfg := config.TLSConfig{
		Enabled: true,
		CAFile:  "/non/existent/ca.crt",
	}
	_, err = CreateTLSConfig(cfg)
	assert.Error(t, err, "Expected error with non-existent CA file")

	// Test with invalid CA file content
	invalidCAFile := filepath.Join(testDir, "invalid.ca")
	err = os.WriteFile(invalidCAFile, []byte("NOT A CERTIFICATE"), 0600)
	require.NoError(t, err)

	cfg = config.TLSConfig{
		Enabled: true,
		CAFile:  invalidCAFile,
	}
	_, err = CreateTLSConfig(cfg)
	assert.Error(t, err, "Expected error with invalid CA file content")

	// Test with valid CA file but invalid certificate files
	cfg = config.TLSConfig{
		Enabled:  true,
		CAFile:   caFile,
		CertFile: "/non/existent/cert.crt",
		KeyFile:  keyFile,
	}
	_, err = CreateTLSConfig(cfg)
	assert.Error(t, err, "Expected error with non-existent cert file")

	cfg = config.TLSConfig{
		Enabled:  true,
		CAFile:   caFile,
		CertFile: certFile,
		KeyFile:  "/non/existent/key.key",
	}
	_, err = CreateTLSConfig(cfg)
	assert.Error(t, err, "Expected error with non-existent key file")

	// Note: We can't easily test a successful certificate loading scenario in a unit test
	// without creating valid certificate files, which requires external resources
}

func TestCreateTLSConfigFromEnv(t *testing.T) {
	// Create test directory for certificates
	testDir, err := os.MkdirTemp("", "tls-test-env")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	// Create dummy certificate and key files for env test
	certFile := filepath.Join(testDir, "env-cert.crt")
	keyFile := filepath.Join(testDir, "env-key.key")

	err = os.WriteFile(certFile, []byte("-----BEGIN CERTIFICATE-----\nMIIDazCCAlOgAwIBAgIUJsw1ius/WyqWrZ6LgYQ6UKTpdGUwDQYJKoZIhvcNAQEL\nBQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM\nGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yMjA1MDEwMDAwMDBaFw0yMzA1\nMDEwMDAwMDBaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw\nHwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3DQEB\nAQUAA4IBDwAwggEKAoIBAQDQB7UQpOyqVjFUTLf3DQmtLLnIVrESL8jM6o7MFt4H\nrwflqVHQOesX6eMGhfJjQJuB3fENo4IpXLnHWuKIiTdqwc1h6Dlc85i8mCzxkf6X\nApFXU+Iq/sRH5XQTJgXiwm4YYXn5LAV7uSvVkf5YVxC3iUzCuzQGwhJrWxQtBkP+\nzilLKrbr5NzL0XGkHWzZVdo5q8i17eUEYcVnEbKvYo5JLgYlm1P4KZAX6nKqMg6h\ndhOYwVp1xyFLp6h8+CKGCgjUKpWW2jku0QUjxCj5hvOQaPoZVmb"), 0600)
	require.NoError(t, err)

	err = os.WriteFile(keyFile, []byte("-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDQB7UQpOyqVjFU\nTLf3DQmtLLnIVrESL8jM6o7MFt4HrwflqVHQOesX6eMGhfJjQJuB3fENo4IpXLnH\nWuKIiTdqwc1h6Dlc85i8mCzxkf6XApFXU+Iq/sRH5XQTJgXiwm4YYXn5LAV7uSvV\nkf5YVxC3iUzCuzQGwhJrWxQtBkP+zilLKrbr5NzL0XGkHWzZVdo5q8i17eUEYcVn\nEbKvYo5JLgYlm1P4KZAX6nKqMg6hdhOYwVp1xyFLp6h8+CKGCgjUKpWW2jku0QUj\nxCj5hvOQaPoZVmb"), 0600)
	require.NoError(t, err)

	// Save current environment to restore later
	oldTLSEnabled := os.Getenv("TAILPOST_TLS_ENABLED")
	oldTLSCertFile := os.Getenv("TAILPOST_TLS_CERT_FILE")
	oldTLSKeyFile := os.Getenv("TAILPOST_TLS_KEY_FILE")
	oldTLSCAFile := os.Getenv("TAILPOST_TLS_CA_FILE")
	oldTLSMinVersion := os.Getenv("TAILPOST_TLS_MIN_VERSION")
	oldTLSMaxVersion := os.Getenv("TAILPOST_TLS_MAX_VERSION")
	oldTLSServerName := os.Getenv("TAILPOST_TLS_SERVER_NAME")
	oldTLSInsecureSkipVerify := os.Getenv("TAILPOST_TLS_INSECURE_SKIP_VERIFY")
	oldTLSPreferServerCipherSuites := os.Getenv("TAILPOST_TLS_PREFER_SERVER_CIPHER_SUITES")

	defer func() {
		os.Setenv("TAILPOST_TLS_ENABLED", oldTLSEnabled)
		os.Setenv("TAILPOST_TLS_CERT_FILE", oldTLSCertFile)
		os.Setenv("TAILPOST_TLS_KEY_FILE", oldTLSKeyFile)
		os.Setenv("TAILPOST_TLS_CA_FILE", oldTLSCAFile)
		os.Setenv("TAILPOST_TLS_MIN_VERSION", oldTLSMinVersion)
		os.Setenv("TAILPOST_TLS_MAX_VERSION", oldTLSMaxVersion)
		os.Setenv("TAILPOST_TLS_SERVER_NAME", oldTLSServerName)
		os.Setenv("TAILPOST_TLS_INSECURE_SKIP_VERIFY", oldTLSInsecureSkipVerify)
		os.Setenv("TAILPOST_TLS_PREFER_SERVER_CIPHER_SUITES", oldTLSPreferServerCipherSuites)
	}()

	// Clear all environment variables
	os.Unsetenv("TAILPOST_TLS_ENABLED")
	os.Unsetenv("TAILPOST_TLS_CERT_FILE")
	os.Unsetenv("TAILPOST_TLS_KEY_FILE")
	os.Unsetenv("TAILPOST_TLS_CA_FILE")
	os.Unsetenv("TAILPOST_TLS_MIN_VERSION")
	os.Unsetenv("TAILPOST_TLS_MAX_VERSION")
	os.Unsetenv("TAILPOST_TLS_SERVER_NAME")
	os.Unsetenv("TAILPOST_TLS_INSECURE_SKIP_VERIFY")
	os.Unsetenv("TAILPOST_TLS_PREFER_SERVER_CIPHER_SUITES")

	// Test with TLS disabled by default
	tlsConfig, err := CreateTLSConfigFromEnv()
	if err != nil {
		t.Errorf("Unexpected error with unset TLS_ENABLED: %v", err)
	}
	if tlsConfig != nil {
		t.Errorf("Expected nil TLS config when TLS_ENABLED is unset, got non-nil")
	}

	// Test with TLS enabled via environment but without cert/key files
	// This is a valid configuration since TLS client config doesn't necessarily need certificates
	os.Setenv("TAILPOST_TLS_ENABLED", "true")
	os.Setenv("TAILPOST_TLS_MIN_VERSION", "tls12")
	os.Setenv("TAILPOST_TLS_SERVER_NAME", "example.com")
	os.Setenv("TAILPOST_TLS_INSECURE_SKIP_VERIFY", "true")

	tlsConfig, err = CreateTLSConfigFromEnv()
	if err != nil {
		t.Errorf("Unexpected error with TLS enabled via env: %v", err)
	}
	if tlsConfig == nil {
		t.Fatalf("Expected non-nil TLS config, got nil")
	}
	assert.Equal(t, "example.com", tlsConfig.ServerName)
	assert.Equal(t, uint16(tls.VersionTLS12), tlsConfig.MinVersion)
	assert.True(t, tlsConfig.InsecureSkipVerify)

	// Test with invalid boolean value
	os.Setenv("TAILPOST_TLS_ENABLED", "not-a-bool")
	_, err = CreateTLSConfigFromEnv()
	if err == nil {
		t.Errorf("Expected error with invalid TLS_ENABLED value, got nil")
	}

	// Test with various environment settings, but still no cert/key since we can't easily create valid ones in tests
	os.Setenv("TAILPOST_TLS_ENABLED", "true")
	os.Setenv("TAILPOST_TLS_SERVER_NAME", "server1.example.com")
	os.Setenv("TAILPOST_TLS_MIN_VERSION", "tls12")
	os.Setenv("TAILPOST_TLS_MAX_VERSION", "tls13")
	os.Setenv("TAILPOST_TLS_INSECURE_SKIP_VERIFY", "true")
	os.Setenv("TAILPOST_TLS_PREFER_SERVER_CIPHER_SUITES", "true")

	tlsConfig, err = CreateTLSConfigFromEnv()
	if err != nil {
		t.Errorf("Unexpected error with complete env config: %v", err)
	}

	assert.NotNil(t, tlsConfig, "TLS config should not be nil")
	assert.Equal(t, "server1.example.com", tlsConfig.ServerName, "Server name should match env value")
	assert.Equal(t, uint16(tls.VersionTLS12), tlsConfig.MinVersion, "Min version should match tls12")
	assert.Equal(t, uint16(tls.VersionTLS13), tlsConfig.MaxVersion, "Max version should match tls13")
	assert.True(t, tlsConfig.InsecureSkipVerify, "InsecureSkipVerify should be true")
	assert.True(t, tlsConfig.PreferServerCipherSuites, "PreferServerCipherSuites should be true")

	// Test disabling PreferServerCipherSuites explicitly
	os.Setenv("TAILPOST_TLS_PREFER_SERVER_CIPHER_SUITES", "false")
	tlsConfig, err = CreateTLSConfigFromEnv()
	if err != nil {
		t.Errorf("Unexpected error with PREFER_SERVER_CIPHER_SUITES=false: %v", err)
	}

	assert.NotNil(t, tlsConfig, "TLS config should not be nil")
	assert.False(t, tlsConfig.PreferServerCipherSuites, "PreferServerCipherSuites should be false")
}

func TestTLSVersionMapping(t *testing.T) {
	// Test that all TLS versions are mapped correctly
	assert.Equal(t, uint16(tls.VersionTLS10), TLSVersion["tls10"], "TLS 1.0 mapping is incorrect")
	assert.Equal(t, uint16(tls.VersionTLS11), TLSVersion["tls11"], "TLS 1.1 mapping is incorrect")
	assert.Equal(t, uint16(tls.VersionTLS12), TLSVersion["tls12"], "TLS 1.2 mapping is incorrect")
	assert.Equal(t, uint16(tls.VersionTLS13), TLSVersion["tls13"], "TLS 1.3 mapping is incorrect")

	// Test case insensitivity with mixed case
	cfg := config.TLSConfig{
		Enabled:    true,
		MinVersion: "TLS12",
	}

	tlsConfig, err := CreateTLSConfig(cfg)
	assert.NoError(t, err, "Should accept mixed case TLS version")
	assert.Equal(t, uint16(tls.VersionTLS12), tlsConfig.MinVersion, "Should handle mixed case TLS version")
}
