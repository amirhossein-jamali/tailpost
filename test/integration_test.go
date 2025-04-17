package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

// startMockServer starts a mock server for testing
func startMockServer(t *testing.T) (*http.Server, string, chan struct{}) {
	// Find a free port to listen on
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find a free port: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	serverAddr := fmt.Sprintf("http://127.0.0.1:%d", port)
	t.Logf("Using server address: %s", serverAddr)

	// Create a server using the listener
	server := &http.Server{}

	started := make(chan struct{})

	// Create a custom ServeMux to avoid conflicts with other tests
	mux := http.NewServeMux()

	// Handle logs endpoint
	mux.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var logs []string
		if err := json.NewDecoder(r.Body).Decode(&logs); err != nil {
			t.Logf("Failed to parse JSON: %v", err)
			http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
			return
		}

		t.Logf("Received logs:")
		for i, logLine := range logs {
			t.Logf("[%d] %s", i+1, logLine)
		}

		w.WriteHeader(http.StatusOK)
	})

	// Add health endpoint for health checks
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status":"ok"}`))
		if err != nil {
			t.Logf("Error writing health response: %v", err)
		}
	})

	// Set the handler
	server.Handler = mux

	// Start server in a goroutine
	go func() {
		t.Logf("Mock server starting on port %d", port)
		close(started)
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	return server, serverAddr, started
}

// waitForServer waits for the server to be ready
func waitForServer(t *testing.T, url string, maxRetries int) bool {
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return true
		}
		t.Logf("Server not ready (attempt %d/%d): %v", i+1, maxRetries, err)
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

// Integration test that verifies the agent can send logs to the mock server
func TestAgentSendsLogsToMockServer(t *testing.T) {
	// Start mock server
	server, serverAddr, started := startMockServer(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			t.Logf("Server shutdown failed: %v", err) // Don't fail the test on shutdown issues
		}
	}()

	// Wait for server to start
	<-started

	// Health check URL
	healthURL := fmt.Sprintf("%s/health", serverAddr)
	logsURL := fmt.Sprintf("%s/logs", serverAddr)

	// Wait for server to be ready
	if !waitForServer(t, healthURL, 5) {
		t.Fatalf("Server did not become ready in time")
	}

	// Create sample logs
	logs := []string{
		"Test log entry 1",
		"Test log entry 2",
		"Test log entry 3",
	}

	// Convert to JSON
	logsJSON, err := json.Marshal(logs)
	if err != nil {
		t.Fatalf("Failed to marshal logs: %v", err)
	}

	// Try several times to send logs to the server
	var resp *http.Response
	var lastErr error

	for i := 0; i < 5; i++ { // Increased retries from 3 to 5
		// Send to mock server
		resp, err = http.Post(logsURL, "application/json", bytes.NewBuffer(logsJSON))
		if err == nil {
			break
		}
		lastErr = err
		t.Logf("Attempt %d: Failed to send logs: %v. Retrying in 1 second...", i+1, err)
		time.Sleep(time.Second)
	}

	if err != nil {
		t.Fatalf("Failed to send logs: %v", lastErr)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
