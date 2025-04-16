//go:build darwin
// +build darwin

package reader

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// Store original macosExecCommand to reset after tests
var originalMacosExecCommand = macosExecCommand

// MockCmd provides a mockable command for testing
type MockCmd struct {
	startFunc func() error
	waitFunc  func() error

	stdout io.ReadCloser
}

func (m *MockCmd) StdoutPipe() (io.ReadCloser, error) {
	return m.stdout, nil
}

func (m *MockCmd) Start() error {
	return m.startFunc()
}

func (m *MockCmd) Wait() error {
	return m.waitFunc()
}

func (m *MockCmd) Process() *os.Process {
	return nil
}

// ResetMacOSExecCommand resets the macosExecCommand variable to the original exec.Command
func ResetMacOSExecCommand() {
	macosExecCommand = originalMacosExecCommand
}

// MockMacOSExecCommand mocks macosExecCommand
func MockMacOSExecCommand(outputStr string, startErr, waitErr error) func(string, ...string) *exec.Cmd {
	return func(command string, args ...string) *exec.Cmd {
		cmd := &MockCmd{
			startFunc: func() error { return startErr },
			waitFunc:  func() error { return waitErr },
			stdout:    io.NopCloser(strings.NewReader(outputStr)),
		}
		return (*exec.Cmd)(reflect.ValueOf(cmd).Addr().Interface().(*reflect.Value).Interface().(*exec.Cmd))
	}
}

// TestMacOSLogReaderInterface checks that MacOSLogReader implements the LogReader interface
func TestMacOSLogReaderInterface(t *testing.T) {
	// Compile-time check that MacOSLogReader implements LogReader
	var _ LogReader = (*MacOSLogReader)(nil)
}

// TestNewMacOSLogReader tests creation of the macOS log reader
func TestNewMacOSLogReader(t *testing.T) {
	testCases := []struct {
		name  string
		query string
	}{
		{
			name:  "Empty query",
			query: "",
		},
		{
			name:  "Process query",
			query: "process == \"kernel\"",
		},
		{
			name:  "Subsystem query",
			query: "subsystem == \"com.apple.system\"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader, err := NewMacOSLogReader(tc.query)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if reader.query != tc.query {
				t.Errorf("Expected query %s, got %s", tc.query, reader.query)
			}
		})
	}
}

// TestMacOSLogReaderStartStop tests the Start and Stop methods
func TestMacOSLogReaderStartStop(t *testing.T) {
	// Skip if actually running the 'log' command would be disruptive
	t.Skip("Skipping test that would start the actual 'log' command")

	reader, err := NewMacOSLogReader("process == \"kernel\"")
	if err != nil {
		t.Fatalf("Error creating reader: %v", err)
	}

	// Start the reader
	err = reader.Start()
	if err != nil {
		t.Fatalf("Error starting reader: %v", err)
	}

	// Verify it's running
	if !reader.running {
		t.Error("Reader should be running after Start()")
	}

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	// Stop the reader
	reader.Stop()

	// Verify the reader has stopped by checking if the stoppedCh is closed
	select {
	case <-reader.stoppedCh:
		// Channel is closed, which is what we expect
	case <-time.After(1 * time.Second):
		t.Fatalf("Reader did not stop within timeout")
	}

	// Verify it's not running
	if reader.running {
		t.Error("Reader should not be running after Stop()")
	}
}

// TestMacOSLogReaderLines verifies the Lines method
func TestMacOSLogReaderLines(t *testing.T) {
	reader, _ := NewMacOSLogReader("process == \"kernel\"")

	lines := reader.Lines()
	if lines == nil {
		t.Error("Lines() returned nil channel")
	}
}

// --- NEW TESTS BELOW ---

// createMockReadCloser creates a ReadCloser for mocking
func createMockReadCloser(content string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(content))
}

// TestMacOSLogReaderCommandArgs tests that the log command is constructed with correct arguments
func TestMacOSLogReaderCommandArgs(t *testing.T) {
	// Save original exec.Command
	originalExecCommand := macosExecCommand
	defer func() { macosExecCommand = originalExecCommand }()

	// Create a mock command execution that captures the arguments
	var capturedArgs []string
	macosExecCommand = func(command string, args ...string) *exec.Cmd {
		capturedArgs = args
		// Return a command that does nothing when started
		return originalExecCommand("echo", "mock")
	}

	testCases := []struct {
		name          string
		query         string
		expectedArgs  []string
		skipActualRun bool
	}{
		{
			name:         "Empty query",
			query:        "",
			expectedArgs: []string{"stream", "--style", "syslog"},
		},
		{
			name:         "Custom query",
			query:        "process == \"kernel\"",
			expectedArgs: []string{"stream", "--predicate", "process == \"kernel\"", "--style", "syslog"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader, _ := NewMacOSLogReader(tc.query)

			// Start without actually running the command
			if tc.skipActualRun {
				t.Skip("Skipping actual execution")
			}

			// Mock the StdoutPipe and Start to avoid actual execution
			// This is a bit of a hack as we can't fully mock exec.Command
			reader.Start()

			// Stop immediately
			reader.Stop()

			// Verify the arguments
			if !reflect.DeepEqual(capturedArgs, tc.expectedArgs) {
				t.Errorf("Expected args: %v, got: %v", tc.expectedArgs, capturedArgs)
			}
		})
	}
}

// TestMacOSLogReaderReadLogs tests the log reading functionality
func TestMacOSLogReaderReadLogs(t *testing.T) {
	mockLogs := `2023-01-01 12:00:01 kernel[0]: This is log line 1
2023-01-01 12:00:02 kernel[0]: This is log line 2
2023-01-01 12:00:03 kernel[0]: This is log line 3
`

	reader, _ := NewMacOSLogReader("process == \"kernel\"")

	// Start the reading process manually
	go reader.readLogs(createMockReadCloser(mockLogs))

	// Read 3 lines with timeout
	receivedLines := []string{}
	timeout := time.After(1 * time.Second)
	for i := 0; i < 3; i++ {
		select {
		case line := <-reader.Lines():
			receivedLines = append(receivedLines, line)
		case <-timeout:
			t.Fatalf("Timeout waiting for log line %d", i+1)
		}
	}

	// Check that we got all the expected lines
	expected := []string{
		"2023-01-01 12:00:01 kernel[0]: This is log line 1",
		"2023-01-01 12:00:02 kernel[0]: This is log line 2",
		"2023-01-01 12:00:03 kernel[0]: This is log line 3",
	}

	for i, exp := range expected {
		if i >= len(receivedLines) {
			t.Fatalf("Missing expected line: %s", exp)
		}
		if receivedLines[i] != exp {
			t.Errorf("Line %d: Expected %q, got %q", i, exp, receivedLines[i])
		}
	}

	// Clean up
	reader.Stop()
}

// TestMacOSLogReaderEmptyLines tests handling of empty lines
func TestMacOSLogReaderEmptyLines(t *testing.T) {
	// Log with some empty lines mixed in
	mockLogs := `
2023-01-01 12:00:01 kernel[0]: This is log line 1

2023-01-01 12:00:02 kernel[0]: This is log line 2

`
	reader, _ := NewMacOSLogReader("process == \"kernel\"")

	// Start the reading process manually
	go reader.readLogs(createMockReadCloser(mockLogs))

	// Read 2 lines with timeout (empty lines should be skipped)
	receivedLines := []string{}
	timeout := time.After(1 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case line := <-reader.Lines():
			receivedLines = append(receivedLines, line)
		case <-timeout:
			break
		}
	}

	// Check that we got all the expected lines and empty lines were skipped
	expected := []string{
		"2023-01-01 12:00:01 kernel[0]: This is log line 1",
		"2023-01-01 12:00:02 kernel[0]: This is log line 2",
	}

	if len(receivedLines) != len(expected) {
		t.Fatalf("Expected %d lines, got %d", len(expected), len(receivedLines))
	}

	for i, exp := range expected {
		if receivedLines[i] != exp {
			t.Errorf("Line %d: Expected %q, got %q", i, exp, receivedLines[i])
		}
	}

	// Clean up
	reader.Stop()
}

// TestMacOSLogReaderStartTwice tests that starting the reader twice doesn't cause issues
func TestMacOSLogReaderStartTwice(t *testing.T) {
	reader, _ := NewMacOSLogReader("process == \"kernel\"")

	// Override the command execution to avoid actual execution
	originalExecCommand := macosExecCommand
	macosExecCommand = func(command string, args ...string) *exec.Cmd {
		return originalExecCommand("echo", "mock")
	}

	// Start the first time
	err1 := reader.Start()
	if err1 != nil {
		t.Fatalf("First start failed: %v", err1)
	}

	// Start the second time
	err2 := reader.Start()
	if err2 != nil {
		t.Errorf("Second start should succeed, got error: %v", err2)
	}

	// Clean up
	reader.Stop()

	// Restore original command
	macosExecCommand = originalExecCommand
}

// TestMacOSLogReaderStartError tests handling of start errors
func TestMacOSLogReaderStartError(t *testing.T) {
	reader, _ := NewMacOSLogReader("process == \"kernel\"")

	// Override the command execution to simulate an error
	originalExecCommand := macosExecCommand
	macosExecCommand = func(command string, args ...string) *exec.Cmd {
		cmd := originalExecCommand("test_nonexistent_command", "should_fail")
		return cmd
	}

	// Attempt to start
	err := reader.Start()

	// Should get an error
	if err == nil {
		t.Error("Expected an error when command cannot be started, got nil")
	}

	// Restore original command
	macosExecCommand = originalExecCommand
}

// mockErrorReadCloser simulates a read error
type mockErrorReadCloser struct {
	readCount int
	data      []byte
}

func (m *mockErrorReadCloser) Read(p []byte) (n int, err error) {
	if m.readCount > 0 {
		return 0, errors.New("simulated read error")
	}

	n = copy(p, m.data)
	m.readCount++
	return n, nil
}

func (m *mockErrorReadCloser) Close() error {
	return nil
}

// TestMacOSLogReaderReadError tests handling of read errors
func TestMacOSLogReaderReadError(t *testing.T) {
	reader, _ := NewMacOSLogReader("process == \"kernel\"")

	// Create a reader that will fail after one read
	errReader := &mockErrorReadCloser{
		data: []byte("2023-01-01 12:00:01 kernel[0]: This is log line 1\n"),
	}

	// Start the reading process manually with our error reader
	doneCh := make(chan struct{})
	go func() {
		reader.readLogs(errReader)
		close(doneCh)
	}()

	// Read one successful line
	received := ""
	select {
	case line := <-reader.Lines():
		received = line
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for log line")
	}

	// Verify the line
	expected := "2023-01-01 12:00:01 kernel[0]: This is log line 1"
	if received != expected {
		t.Errorf("Expected %q, got %q", expected, received)
	}

	// The readLogs function should exit due to the error
	select {
	case <-doneCh:
		// Success - readLogs exited
	case <-time.After(1 * time.Second):
		t.Fatal("readLogs did not exit after error")
	}

	// Clean up
	reader.Stop()
}

// TestMacOSLogReaderStopWhileReading tests stopping the reader during reading
func TestMacOSLogReaderStopWhileReading(t *testing.T) {
	// Generate a large amount of log data
	var logBuffer bytes.Buffer
	for i := 0; i < 1000; i++ {
		logBuffer.WriteString(fmt.Sprintf("2023-01-01 12:00:%02d kernel[0]: This is log line %d\n", i%60, i))
	}

	reader, _ := NewMacOSLogReader("process == \"kernel\"")

	// Use a wait group to track when readLogs completes
	var wg sync.WaitGroup
	wg.Add(1)

	// Start reading from our buffer
	go func() {
		defer wg.Done()
		reader.readLogs(io.NopCloser(&logBuffer))
	}()

	// Read a few lines to ensure it's working
	receivedCount := 0
	timeout := time.After(500 * time.Millisecond)
readLoop:
	for {
		select {
		case <-reader.Lines():
			receivedCount++
			if receivedCount >= 5 {
				break readLoop
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for initial log lines, got %d", receivedCount)
		}
	}

	// Stop the reader while it's still processing
	reader.Stop()

	// Wait for readLogs to complete with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - readLogs exited after Stop
	case <-time.After(1 * time.Second):
		t.Fatal("readLogs did not exit after Stop")
	}
}

// TestMacOSLogReaderQueryEscaping tests that query predicates are properly escaped
func TestMacOSLogReaderQueryEscaping(t *testing.T) {
	// Create a reader with a complex query containing quotes and special characters
	query := `process == "app with \"quotes\"" && (level == "error" || level == "fault")`
	reader, _ := NewMacOSLogReader(query)

	// Save original exec.Command
	originalExecCommand := macosExecCommand
	defer func() { macosExecCommand = originalExecCommand }()

	// Create a mock command execution that captures the arguments
	var capturedArgs []string
	macosExecCommand = func(command string, args ...string) *exec.Cmd {
		capturedArgs = args
		// Return a command that does nothing when started
		return originalExecCommand("echo", "mock")
	}

	// Start without actually running the command
	reader.Start()
	reader.Stop()

	// Verify the predicate argument contains our query exactly as provided
	expectedPredicate := query
	foundPredicate := false

	for i := 0; i < len(capturedArgs)-1; i++ {
		if capturedArgs[i] == "--predicate" && capturedArgs[i+1] == expectedPredicate {
			foundPredicate = true
			break
		}
	}

	if !foundPredicate {
		t.Errorf("Query not properly passed to command. Expected predicate %q not found in args: %v",
			expectedPredicate, capturedArgs)
	}
}
