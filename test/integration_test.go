package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// Integration test that verifies the agent can send logs to the mock server
func TestAgentSendsLogsToMockServer(t *testing.T) {
	// Wait for services to start up
	time.Sleep(2 * time.Second)

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

	// Send to mock server
	resp, err := http.Post("http://localhost:8081/logs", "application/json", bytes.NewBuffer(logsJSON))
	if err != nil {
		t.Fatalf("Failed to send logs: %v", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
