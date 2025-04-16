//go:build windows
// +build windows

package reader

import (
	"strings"
	"testing"
	"time"
)

// TestWindowsEventLogReaderInterface checks that WindowsEventLogReader implements the LogReader interface
func TestWindowsEventLogReaderInterface(t *testing.T) {
	// Compile-time check that WindowsEventLogReader implements LogReader
	var _ LogReader = (*WindowsEventLogReader)(nil)
}

// TestNewWindowsEventLogReader tests creation of the windows event log reader
func TestNewWindowsEventLogReader(t *testing.T) {
	testCases := []struct {
		name          string
		logName       string
		level         string
		expectLogName string
		expectLevel   EventLogLevel
	}{
		{
			name:          "Default values",
			logName:       "",
			level:         "",
			expectLogName: "Application",
			expectLevel:   EventLogLevelInformation,
		},
		{
			name:          "Custom values",
			logName:       "System",
			level:         "Error",
			expectLogName: "System",
			expectLevel:   EventLogLevelError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader, err := NewWindowsEventLogReader(tc.logName, tc.level)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if reader.logName != tc.expectLogName {
				t.Errorf("Expected log name %s, got %s", tc.expectLogName, reader.logName)
			}

			if reader.minLevel != tc.expectLevel {
				t.Errorf("Expected level %s, got %s", tc.expectLevel, reader.minLevel)
			}
		})
	}
}

// TestWindowsEventLogReaderStartStop tests the Start and Stop methods
func TestWindowsEventLogReaderStartStop(t *testing.T) {
	reader, err := NewWindowsEventLogReader("Application", "Information")
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

// TestWindowsEventFormatting tests the event formatting
func TestWindowsEventFormatting(t *testing.T) {
	event := &windowsEvent{
		TimeCreated: "2023-01-01T12:00:00Z",
		Source:      "TestSource",
		EventID:     1000,
		Level:       "Information",
		Computer:    "DESKTOP-TEST",
		RecordID:    12345,
		Message:     "Test message\nwith newline",
	}

	formatted := event.formatAsLogLine()
	expected := "[2023-01-01T12:00:00Z] TestSource EventID=1000 Computer=DESKTOP-TEST Provider=TestSource Level=Information RecordID=12345 Message=Test message with newline"

	if formatted != expected {
		t.Errorf("Expected formatted event:\n%s\nGot:\n%s", expected, formatted)
	}
}

// TestMockEventGeneration tests that getLatestEvents returns mock events as expected
func TestMockEventGeneration(t *testing.T) {
	reader, _ := NewWindowsEventLogReader("Application", "Information")

	events, err := reader.getLatestEvents(0)
	if err != nil {
		t.Fatalf("Got unexpected error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.Source != "Application" {
		t.Errorf("Expected source 'Application', got '%s'", event.Source)
	}

	if event.Level != "Information" {
		t.Errorf("Expected level 'Information', got '%s'", event.Level)
	}

	if event.RecordID != 1 {
		t.Errorf("Expected RecordID 1, got %d", event.RecordID)
	}
}

// TestWindowsEventLogReaderLines verifies the Lines method
func TestWindowsEventLogReaderLines(t *testing.T) {
	reader, _ := NewWindowsEventLogReader("Application", "Information")

	lines := reader.Lines()
	if lines == nil {
		t.Error("Lines() returned nil channel")
	}
}

// --- NEW TESTS BELOW ---

// TestWindowsEventLogLevels tests the different event log levels
func TestWindowsEventLogLevels(t *testing.T) {
	testCases := []struct {
		name  string
		level string
	}{
		{"Information", string(EventLogLevelInformation)},
		{"Warning", string(EventLogLevelWarning)},
		{"Error", string(EventLogLevelError)},
		{"Critical", string(EventLogLevelCritical)},
		{"Verbose", string(EventLogLevelVerbose)},
		{"Custom", "Custom"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader, err := NewWindowsEventLogReader("Application", tc.level)
			if err != nil {
				t.Fatalf("Failed to create reader with level %s: %v", tc.level, err)
			}

			if string(reader.minLevel) != tc.level {
				// If level is empty, it should default to Information
				if tc.level == "" && reader.minLevel == EventLogLevelInformation {
					return
				}
				t.Errorf("Expected level %s, got %s", tc.level, reader.minLevel)
			}

			// Verify the level is used when generating events
			events, err := reader.getLatestEvents(0)
			if err != nil {
				t.Fatalf("Failed to get events: %v", err)
			}

			if len(events) > 0 {
				event := events[0]
				expectedLevel := tc.level
				if tc.level == "" {
					expectedLevel = string(EventLogLevelInformation)
				}
				if event.Level != expectedLevel {
					t.Errorf("Expected event level %s, got %s", expectedLevel, event.Level)
				}
			}
		})
	}
}

// TestWindowsEventLogReaderStartTwice tests calling Start twice
func TestWindowsEventLogReaderStartTwice(t *testing.T) {
	reader, err := NewWindowsEventLogReader("Application", "Information")
	if err != nil {
		t.Fatalf("Error creating reader: %v", err)
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

	// Stop the reader
	reader.Stop()
}

// TestWindowsEventLogReaderStopWithoutStart tests stopping without starting
func TestWindowsEventLogReaderStopWithoutStart(t *testing.T) {
	reader, _ := NewWindowsEventLogReader("Application", "Information")

	// Should not panic or block
	reader.Stop()

	// Verify it's not running
	if reader.running {
		t.Error("Reader should not be running after Stop()")
	}
}

// TestWindowsEventLogReaderReadEvents tests reading multiple events
func TestWindowsEventLogReaderReadEvents(t *testing.T) {
	reader, _ := NewWindowsEventLogReader("Application", "Information")

	// Start the reader
	err := reader.Start()
	if err != nil {
		t.Fatalf("Failed to start reader: %v", err)
	}

	// Let it run long enough to generate a few events
	// The mock implementation generates events on a ticker
	timeout := time.After(10 * time.Second)
	count := 0
	expectedMinimum := 1 // Expect at least one event

	// Read until we get enough events or timeout
readLoop:
	for {
		select {
		case line := <-reader.Lines():
			if strings.Contains(line, "mock Windows event") {
				count++
				if count >= expectedMinimum {
					break readLoop
				}
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for events. Got %d, expected at least %d", count, expectedMinimum)
		}
	}

	// Stop the reader
	reader.Stop()

	// Verify we got at least the expected number of events
	if count < expectedMinimum {
		t.Errorf("Expected at least %d events, got %d", expectedMinimum, count)
	}
}

// TestWindowsEventLogReaderWithDifferentLogNames tests using different log names
func TestWindowsEventLogReaderWithDifferentLogNames(t *testing.T) {
	testCases := []struct {
		name     string
		logName  string
		expected string
	}{
		{"Application", "Application", "Application"},
		{"System", "System", "System"},
		{"Security", "Security", "Security"},
		{"Custom", "CustomLog", "CustomLog"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader, err := NewWindowsEventLogReader(tc.logName, "Information")
			if err != nil {
				t.Fatalf("Failed to create reader with log name %s: %v", tc.logName, err)
			}

			if reader.logName != tc.expected {
				t.Errorf("Expected log name %s, got %s", tc.expected, reader.logName)
			}

			// Verify the log name is used when generating events
			events, err := reader.getLatestEvents(0)
			if err != nil {
				t.Fatalf("Failed to get events: %v", err)
			}

			if len(events) > 0 {
				event := events[0]
				if event.Source != tc.expected {
					t.Errorf("Expected event source %s, got %s", tc.expected, event.Source)
				}
			}
		})
	}
}

// TestWindowsEventLogReaderTracksLastRecordID tests that the reader properly tracks the last record ID
func TestWindowsEventLogReaderTracksLastRecordID(t *testing.T) {
	reader, _ := NewWindowsEventLogReader("Application", "Information")

	// First get events with record ID 0
	events1, err := reader.getLatestEvents(0)
	if err != nil {
		t.Fatalf("Failed to get first events: %v", err)
	}

	if len(events1) == 0 {
		t.Fatalf("Expected at least one event in first batch")
	}

	firstRecordID := events1[0].RecordID

	// Now get events with the record ID from the first event
	events2, err := reader.getLatestEvents(firstRecordID)
	if err != nil {
		t.Fatalf("Failed to get second events: %v", err)
	}

	if len(events2) == 0 {
		t.Fatalf("Expected at least one event in second batch")
	}

	secondRecordID := events2[0].RecordID

	// The second record ID should be higher than the first
	if secondRecordID <= firstRecordID {
		t.Errorf("Expected second record ID (%d) to be higher than first (%d)", secondRecordID, firstRecordID)
	}
}

// TestWindowsEventWithComplexMessage tests formatting events with complex messages
func TestWindowsEventWithComplexMessage(t *testing.T) {
	testCases := []struct {
		name    string
		message string
	}{
		{"With newlines", "Line 1\nLine 2\nLine 3"},
		{"With quotes", "Message with \"quotes\""},
		{"With special chars", "Message with [special] {chars}"},
		{"With tabs", "Message with\ttabs"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := &windowsEvent{
				TimeCreated: "2023-01-01T12:00:00Z",
				Source:      "TestSource",
				EventID:     1000,
				Level:       "Information",
				Computer:    "DESKTOP-TEST",
				RecordID:    12345,
				Message:     tc.message,
			}

			formatted := event.formatAsLogLine()

			// Verify newlines are replaced
			if strings.Contains(formatted, "\n") {
				t.Errorf("Formatted event should not contain newlines: %s", formatted)
			}

			// Verify the message is included
			expectedMessage := strings.ReplaceAll(tc.message, "\n", " ")
			if !strings.Contains(formatted, "Message="+expectedMessage) {
				t.Errorf("Formatted event doesn't contain expected message. Expected to find 'Message=%s' in: %s",
					expectedMessage, formatted)
			}
		})
	}
}
