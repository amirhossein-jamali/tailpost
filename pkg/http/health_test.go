package http

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/amirhossein-jamali/tailpost/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHealthServer(t *testing.T) {
	// Create new health server
	listenAddr := ":8080"
	server := NewHealthServer(listenAddr)

	// Verify server properties
	if server.listenAddr != listenAddr {
		t.Errorf("Expected listen address %s, got %s", listenAddr, server.listenAddr)
	}

	if server.ready {
		t.Errorf("Expected ready to be false, got true")
	}
}

func TestHealthHandler(t *testing.T) {
	// Create new health server
	server := NewHealthServer(":8080")

	// Create a request to pass to our handler
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.healthHandler)

	// Call the handler directly
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the content type
	expectedContentType := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedContentType)
	}

	// Parse the response body as JSON
	var status HealthStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &status); err != nil {
		t.Errorf("Error parsing JSON response: %v", err)
	}

	// Check the status
	if status.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", status.Status)
	}

	// Check that timestamp is a valid time
	_, err = time.Parse(time.RFC3339, status.Timestamp)
	if err != nil {
		t.Errorf("Timestamp is not in the correct format: %s", status.Timestamp)
	}

	// Check the version
	if status.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", status.Version)
	}
}

func TestReadyHandlerWhenNotReady(t *testing.T) {
	// Create new health server (not ready by default)
	server := NewHealthServer(":8080")

	// Create a request to pass to our handler
	req, err := http.NewRequest("GET", "/ready", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.readyHandler)

	// Call the handler directly
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusServiceUnavailable)
	}

	// Parse the response body as JSON
	var status HealthStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &status); err != nil {
		t.Errorf("Error parsing JSON response: %v", err)
	}

	// Check the status
	if status.Status != "not ready" {
		t.Errorf("Expected status 'not ready', got '%s'", status.Status)
	}
}

func TestReadyHandlerWhenReady(t *testing.T) {
	// Create new health server and set to ready
	server := NewHealthServer(":8080")
	server.SetReady(true)

	// Create a request to pass to our handler
	req, err := http.NewRequest("GET", "/ready", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.readyHandler)

	// Call the handler directly
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Parse the response body as JSON
	var status HealthStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &status); err != nil {
		t.Errorf("Error parsing JSON response: %v", err)
	}

	// Check the status
	if status.Status != "ready" {
		t.Errorf("Expected status 'ready', got '%s'", status.Status)
	}
}

func TestMetricsHandler(t *testing.T) {
	// Create new health server
	server := NewHealthServer(":8080")

	// Create a request to pass to our handler
	req, err := http.NewRequest("GET", "/metrics", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.metricsHandler)

	// Call the handler directly
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the content type
	expectedContentType := "text/plain"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Handler returned wrong content type: got %v want %v", contentType, expectedContentType)
	}

	// Check that the response contains expected metrics
	body, err := io.ReadAll(rr.Body)
	if err != nil {
		t.Fatal(err)
	}
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "# HELP tailpost_up") {
		t.Errorf("Metrics missing expected help text")
	}

	if !strings.Contains(bodyStr, "# TYPE tailpost_up gauge") {
		t.Errorf("Metrics missing expected type declaration")
	}

	if !strings.Contains(bodyStr, "tailpost_up 1") {
		t.Errorf("Metrics missing expected value")
	}
}

func TestSetReadyAndIsReady(t *testing.T) {
	// Create new health server
	server := NewHealthServer(":8080")

	// Check initial state
	if server.IsReady() {
		t.Errorf("Expected IsReady() to return false initially")
	}

	// Set ready to true
	server.SetReady(true)
	if !server.IsReady() {
		t.Errorf("Expected IsReady() to return true after SetReady(true)")
	}

	// Set ready to false
	server.SetReady(false)
	if server.IsReady() {
		t.Errorf("Expected IsReady() to return false after SetReady(false)")
	}
}

func TestStartAndStop(t *testing.T) {
	// Create new health server on a random port
	server := NewHealthServer(":0") // Using port 0 lets the OS assign a random free port

	// Start the server
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Stop the server
	err = server.Stop()
	if err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}
}

// Test for concurrent readiness changes
func TestConcurrentReadinessChanges(t *testing.T) {
	server := NewHealthServer(":8080")

	// Number of goroutines for concurrency test
	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // For reading and writing

	// Goroutines that regularly change readiness value
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			ready := idx%2 == 0 // Alternating true and false
			server.SetReady(ready)
		}(i)
	}

	// Goroutines that concurrently read readiness status
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			// Just do a read - the important thing is that it doesn't panic
			_ = server.IsReady()
		}()
	}

	// Wait for all goroutines to finish
	wg.Wait()
	// If we reach here, no panics occurred
}

// Test server behavior with invalid HTTP methods
func TestInvalidHTTPMethod(t *testing.T) {
	server := NewHealthServer(":8080")

	// Test POST method for health endpoint
	req, err := http.NewRequest("POST", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.healthHandler)
	handler.ServeHTTP(rr, req)

	// Even with POST, the server should respond (according to current implementation)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code for POST: got %v want %v", status, http.StatusOK)
	}
}

// Test for when the port is already in use
func TestStartWithUsedPort(t *testing.T) {
	// First occupy a specific port
	listener, err := net.Listen("tcp", ":9090")
	if err != nil {
		t.Fatalf("Could not bind to port for test: %v", err)
	}
	defer listener.Close()

	// Now try to start the server on the same port
	server := NewHealthServer(":9090")

	// Start server should run asynchronously and error is logged in goroutine
	// So start will succeed, but ListenAndServe will error in the goroutine
	err = server.Start()
	if err != nil {
		t.Fatalf("Start unexpectedly returned error: %v", err)
	}

	// Wait a bit to give the goroutine time to run
	time.Sleep(100 * time.Millisecond)

	// Cleanup
	if err := server.Stop(); err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}
}

// Test stopping the server before starting it
func TestStopBeforeStart(t *testing.T) {
	server := NewHealthServer(":8080")

	// Stopping the server before starting should be error-free
	err := server.Stop()
	if err != nil {
		t.Errorf("Stop returned error when server wasn't started: %v", err)
	}
}

// Test to ensure no race conditions in HealthStatus structure
func TestHealthStatusConcurrency(t *testing.T) {
	server := NewHealthServer(":8080")

	const numRequests = 100
	var wg sync.WaitGroup
	wg.Add(numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()

			req, _ := http.NewRequest("GET", "/health", nil)
			rr := httptest.NewRecorder()

			handler := http.HandlerFunc(server.healthHandler)
			handler.ServeHTTP(rr, req)

			// Ensure response is valid JSON
			var status HealthStatus
			err := json.Unmarshal(rr.Body.Bytes(), &status)
			if err != nil {
				t.Errorf("Failed to parse JSON: %v", err)
			}
		}()
	}

	wg.Wait()
}

// Test with invalid path in router
func TestInvalidPath(t *testing.T) {
	// Set up real server to test router
	server := NewHealthServer(":0")
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		if err := server.Stop(); err != nil {
			t.Fatalf("Failed to stop server: %v", err)
		}
	}()

	// Find allocated port
	addr := server.server.Addr
	if addr == ":0" {
		// This isn't a straightforward way to get the allocated port,
		// but for now we can use httptest to test the router
		t.Skip("Skipping actual HTTP request test as we can't determine the port")
	}

	// Create a request to an invalid path
	req, err := http.NewRequest("GET", "/invalid", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	// Build our router with the same settings as server.Start()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", server.healthHandler)
	mux.HandleFunc("/ready", server.readyHandler)
	mux.HandleFunc("/metrics", server.metricsHandler)

	// Send request to router
	mux.ServeHTTP(rr, req)

	// Check status code - should be 404
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Router returned wrong status code for invalid path: got %v want %v", status, http.StatusNotFound)
	}
}

// ----- Tests for security and TLS features -----

// Test NewSecureHealthServer
func TestNewSecureHealthServer(t *testing.T) {
	// TLS settings
	tlsConfig := config.TLSConfig{
		Enabled:  true,
		CertFile: "cert.pem",
		KeyFile:  "key.pem",
	}

	// Auth settings
	authConfig := config.AuthConfig{
		Type:     "basic",
		Username: "test",
		Password: "test123",
	}

	// General security settings
	securityConfig := config.SecurityConfig{
		TLS:  tlsConfig,
		Auth: authConfig,
	}

	// Create secure server
	server, err := NewSecureHealthServer(":8443", securityConfig)
	require.NoError(t, err)
	assert.NotNil(t, server)

	// Check settings
	assert.Equal(t, ":8443", server.listenAddr)
	assert.True(t, server.useTLS)
	assert.Equal(t, "cert.pem", server.certFile)
	assert.Equal(t, "key.pem", server.keyFile)
	assert.NotNil(t, server.authProvider)
}

// Test NewSecureHealthServer with invalid auth
func TestNewSecureHealthServerWithInvalidAuth(t *testing.T) {
	// Settings with invalid auth type
	securityConfig := config.SecurityConfig{
		Auth: config.AuthConfig{
			Type: "unsupported", // Invalid auth type
		},
	}

	// Expect error
	_, err := NewSecureHealthServer(":8443", securityConfig)
	assert.Error(t, err)
}

// Test withAuth with active authentication
func TestWithAuthAuthenticated(t *testing.T) {
	// Create a mock for AuthProvider
	mockAuth := &MockAuthProvider{
		authenticateResult: true,
		authenticateError:  nil,
	}

	server := &HealthServer{
		listenAddr:   ":8080",
		authProvider: mockAuth,
	}

	// Create handler with authentication
	handler := server.withAuth(server.healthHandler)

	// Normal request
	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	// Execute handler
	handler.ServeHTTP(rr, req)

	// Check result - should succeed
	assert.Equal(t, http.StatusOK, rr.Code)
}

// Test withAuth with failed authentication
func TestWithAuthUnauthenticated(t *testing.T) {
	// Create a mock for AuthProvider that rejects authentication
	mockAuth := &MockAuthProvider{
		authenticateResult: false,
		authenticateError:  nil,
	}

	server := &HealthServer{
		listenAddr:   ":8080",
		authProvider: mockAuth,
	}

	// Create handler with authentication
	handler := server.withAuth(server.healthHandler)

	// Normal request
	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	// Execute handler
	handler.ServeHTTP(rr, req)

	// Check result - should get 401 error
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// Test withAuth with error during authentication
func TestWithAuthError(t *testing.T) {
	// Create a mock for AuthProvider that produces an error
	mockAuth := &MockAuthProvider{
		authenticateResult: false,
		authenticateError:  assert.AnError,
	}

	server := &HealthServer{
		listenAddr:   ":8080",
		authProvider: mockAuth,
	}

	// Create handler with authentication
	handler := server.withAuth(server.healthHandler)

	// Normal request
	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	// Execute handler
	handler.ServeHTTP(rr, req)

	// Check result - should get 500 error
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// Test SetTLSConfig
func TestSetTLSConfig(t *testing.T) {
	server := NewHealthServer(":8443")

	// TLS configuration
	customTLS := &tls.Config{
		MinVersion: tls.VersionTLS12, // This is 0x0303 in uint16
	}

	// Start server
	err := server.Start()
	require.NoError(t, err)

	// Set TLS configuration
	server.SetTLSConfig(customTLS)

	// Check that configuration is applied
	assert.Equal(t, uint16(tls.VersionTLS12), server.server.TLSConfig.MinVersion)

	// Cleanup
	if err := server.Stop(); err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}
}

// Test no effect of SetTLSConfig with nil configuration
func TestSetTLSConfigWithNil(t *testing.T) {
	server := NewHealthServer(":8443")

	// Start server
	err := server.Start()
	require.NoError(t, err)

	// First set a non-nil configuration
	customTLS := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	server.SetTLSConfig(customTLS)

	// Now send a nil configuration
	server.SetTLSConfig(nil)

	// Previous configuration should still exist
	assert.NotNil(t, server.server.TLSConfig)

	// Cleanup
	if err := server.Stop(); err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}
}

// Test SetTLSConfig before Start
func TestSetTLSConfigBeforeStart(t *testing.T) {
	server := NewHealthServer(":8443")

	// TLS configuration
	customTLS := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Set TLS configuration before starting
	// Should not cause any changes or errors
	server.SetTLSConfig(customTLS)

	// Check that server is still nil
	assert.Nil(t, server.server)
}

// ----- Helper structures -----

// MockAuthProvider for testing authentication
type MockAuthProvider struct {
	authenticateResult bool
	authenticateError  error
}

func (m *MockAuthProvider) Authenticate(r *http.Request) (bool, error) {
	return m.authenticateResult, m.authenticateError
}

func (m *MockAuthProvider) GetCredentials(r *http.Request) (string, string, error) {
	return "test-user", "test-password", nil
}

// AddAuthentication added to implement the AuthProvider interface completely
func (m *MockAuthProvider) AddAuthentication(req *http.Request) error {
	// No need to change the request, just return nil
	return nil
}
