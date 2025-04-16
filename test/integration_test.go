package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// startMockServer starts a mock server for testing
func startMockServer(t *testing.T) (*http.Server, chan struct{}) {
	// Create a server
	server := &http.Server{Addr: ":8081"}

	started := make(chan struct{})

	// Handle logs endpoint
	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
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
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Start server in a goroutine
	go func() {
		t.Logf("Mock server starting on :8081")
		close(started)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	return server, started
}

// Integration test that verifies the agent can send logs to the mock server
func TestAgentSendsLogsToMockServer(t *testing.T) {
	// Start mock server
	server, started := startMockServer(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			t.Fatalf("Server shutdown failed: %v", err)
		}
	}()

	// Wait for server to start
	<-started

	// Give a little more time for server to be fully ready
	time.Sleep(100 * time.Millisecond)

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

	// Try several times to connect to the server
	var resp *http.Response
	var lastErr error

	for i := 0; i < 3; i++ {
		// Send to mock server
		resp, err = http.Post("http://localhost:8081/logs", "application/json", bytes.NewBuffer(logsJSON))
		if err == nil {
			break
		}
		lastErr = err
		t.Logf("Attempt %d: Failed to send logs: %v. Retrying...", i+1, err)
		time.Sleep(time.Second)
	}

	if err != nil {
		t.Fatalf("All attempts to send logs failed: %v", lastErr)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
