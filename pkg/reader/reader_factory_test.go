package reader

import (
	"runtime"
	"strings"
	"testing"
)

// Mock the container reader for testing
func init() {
	// Replace NewContainerReader with a mock version for testing
	NewContainerReader = func(namespace, podName, containerName string) (LogReader, error) {
		return &mockContainerReader{
			namespace:     namespace,
			podName:       podName,
			containerName: containerName,
			lines:         make(chan string),
		}, nil
	}
}

// mockContainerReader is a simple mock for container reader
type mockContainerReader struct {
	namespace     string
	podName       string
	containerName string
	lines         chan string
}

func (m *mockContainerReader) Start() error {
	return nil
}

func (m *mockContainerReader) Lines() <-chan string {
	return m.lines
}

func (m *mockContainerReader) Stop() {
	close(m.lines)
}

// TestParseSourceType tests the ParseSourceType function
func TestParseSourceType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected LogSourceType
		wantErr  bool
	}{
		{
			name:     "File source type",
			input:    "file",
			expected: FileSourceType,
			wantErr:  false,
		},
		{
			name:     "Container source type",
			input:    "container",
			expected: ContainerSourceType,
			wantErr:  false,
		},
		{
			name:     "Pod source type",
			input:    "pod",
			expected: PodSourceType,
			wantErr:  false,
		},
		{
			name:     "Windows event log source type",
			input:    "windows_event",
			expected: WindowsEventSourceType,
			wantErr:  false,
		},
		{
			name:     "Windows short alias",
			input:    "windows",
			expected: WindowsEventSourceType,
			wantErr:  false,
		},
		{
			name:     "Event alias",
			input:    "event",
			expected: WindowsEventSourceType,
			wantErr:  false,
		},
		{
			name:     "macOS ASL source type",
			input:    "macos_asl",
			expected: MacOSASLSourceType,
			wantErr:  false,
		},
		{
			name:     "macOS short alias",
			input:    "macos",
			expected: MacOSASLSourceType,
			wantErr:  false,
		},
		{
			name:     "ASL alias",
			input:    "asl",
			expected: MacOSASLSourceType,
			wantErr:  false,
		},
		{
			name:     "Invalid source type",
			input:    "invalid",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceType, err := ParseSourceType(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if sourceType != tt.expected {
					t.Errorf("Expected source type %s, got %s", tt.expected, sourceType)
				}
			}
		})
	}
}

// TestNewReader tests the NewReader function
func TestNewReader(t *testing.T) {
	tests := []struct {
		name    string
		config  LogSourceConfig
		wantErr bool
		skipOS  string
	}{
		{
			name: "File reader",
			config: LogSourceConfig{
				Type: FileSourceType,
				Path: "/tmp/test.log",
			},
			wantErr: false,
		},
		{
			name: "File reader - missing path",
			config: LogSourceConfig{
				Type: FileSourceType,
			},
			wantErr: true,
		},
		{
			name: "Container reader",
			config: LogSourceConfig{
				Type:          ContainerSourceType,
				Namespace:     "default",
				PodName:       "test-pod",
				ContainerName: "test-container",
			},
			wantErr: false,
		},
		{
			name: "Container reader - missing namespace",
			config: LogSourceConfig{
				Type:          ContainerSourceType,
				PodName:       "test-pod",
				ContainerName: "test-container",
			},
			wantErr: true,
		},
		{
			name: "Pod reader",
			config: LogSourceConfig{
				Type:        PodSourceType,
				PodSelector: "app=test",
			},
			wantErr: true, // Not implemented yet
		},
		{
			name: "Windows event log reader",
			config: LogSourceConfig{
				Type:                WindowsEventSourceType,
				WindowsEventLogName: "Application",
			},
			wantErr: runtime.GOOS != "windows",
			skipOS: func() string {
				if runtime.GOOS != "windows" {
					return ""
				} else {
					return "linux,darwin"
				}
			}(),
		},
		{
			name: "macOS log reader",
			config: LogSourceConfig{
				Type:          MacOSASLSourceType,
				MacOSLogQuery: "process == \"kernel\"",
			},
			wantErr: runtime.GOOS != "darwin",
			skipOS: func() string {
				if runtime.GOOS != "darwin" {
					return ""
				} else {
					return "linux,windows"
				}
			}(),
		},
		{
			name: "Unknown reader type",
			config: LogSourceConfig{
				Type: "unknown",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		// Skip tests that are not applicable to the current OS
		if tt.skipOS != "" && ((runtime.GOOS == "linux" && tt.skipOS == "linux") ||
			(runtime.GOOS == "windows" && tt.skipOS == "windows") ||
			(runtime.GOOS == "darwin" && tt.skipOS == "darwin")) {
			continue
		}

		t.Run(tt.name, func(t *testing.T) {
			reader, err := NewReader(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if reader == nil {
					t.Errorf("Expected reader but got nil")
				}
			}
		})
	}
}

// TestLogSourceTypeString tests the string representation of LogSourceType
func TestLogSourceTypeString(t *testing.T) {
	tests := []struct {
		sourceType LogSourceType
		expected   string
	}{
		{FileSourceType, "file"},
		{ContainerSourceType, "container"},
		{PodSourceType, "pod"},
		{WindowsEventSourceType, "windows_event"},
		{MacOSASLSourceType, "macos_asl"},
		{LogSourceType("custom"), "custom"},
	}

	for _, test := range tests {
		t.Run(string(test.sourceType), func(t *testing.T) {
			if string(test.sourceType) != test.expected {
				t.Errorf("Expected string representation to be %s, got %s",
					test.expected, string(test.sourceType))
			}
		})
	}
}

// TestPlatformSpecificReaders tests the platform-specific reader wrappers
func TestPlatformSpecificReaders(t *testing.T) {
	// Test macOS log reader wrapper
	t.Run("macOS log reader wrapper", func(t *testing.T) {
		reader, err := newMacOSLogReader("test query")

		if runtime.GOOS == "darwin" {
			// On macOS, this should be overridden by the init function in macos_log_reader.go
			// So we can't really test the implementation here, but we can check that we don't get an error
			if err != nil {
				t.Errorf("Unexpected error on macOS: %v", err)
			}
		} else {
			// On non-macOS platforms, we should get an error
			if err == nil {
				t.Errorf("Expected error on non-macOS platform but got nil")
			}
			if reader != nil {
				t.Errorf("Expected nil reader on non-macOS platform but got %v", reader)
			}
		}
	})

	// Test Windows event log reader wrapper
	t.Run("Windows event log reader wrapper", func(t *testing.T) {
		reader, err := newWindowsEventLogReader("Application", "Information")

		if runtime.GOOS == "windows" {
			// On Windows, this should be overridden by the init function in windows_event_reader.go
			// So we can't really test the implementation here, but we can check that we don't get an error
			if err != nil {
				t.Errorf("Unexpected error on Windows: %v", err)
			}
		} else {
			// On non-Windows platforms, we should get an error
			if err == nil {
				t.Errorf("Expected error on non-Windows platform but got nil")
			}
			if reader != nil {
				t.Errorf("Expected nil reader on non-Windows platform but got %v", reader)
			}
		}
	})
}

// Helper function to get the type of a reader for testing
func GetReaderType(reader LogReader) string {
	switch reader.(type) {
	case *FileReader:
		return "*reader.FileReader"
	case *ContainerReader:
		return "*reader.ContainerReader"
	default:
		return "unknown"
	}
}

// --- NEW TESTS BELOW ---

// TestParseSourceTypeCaseSensitivity tests that source type parsing is case-insensitive
func TestParseSourceTypeCaseSensitivity(t *testing.T) {
	testCases := []struct {
		input          string
		expectedOutput LogSourceType
	}{
		{"FILE", FileSourceType},
		{"File", FileSourceType},
		{"CONTAINER", ContainerSourceType},
		{"Container", ContainerSourceType},
		{"POD", PodSourceType},
		{"Pod", PodSourceType},
		{"WINDOWS_EVENT", WindowsEventSourceType},
		{"Windows_Event", WindowsEventSourceType},
		{"WINDOWS", WindowsEventSourceType},
		{"Windows", WindowsEventSourceType},
		{"EVENT", WindowsEventSourceType},
		{"Event", WindowsEventSourceType},
		{"MACOS_ASL", MacOSASLSourceType},
		{"MacOS_ASL", MacOSASLSourceType},
		{"MACOS", MacOSASLSourceType},
		{"MacOS", MacOSASLSourceType},
		{"ASL", MacOSASLSourceType},
		{"Asl", MacOSASLSourceType},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result, err := ParseSourceType(tc.input)
			if err != nil {
				t.Fatalf("ParseSourceType(%q) returned error: %v", tc.input, err)
			}
			if result != tc.expectedOutput {
				t.Errorf("ParseSourceType(%q) = %q, want %q", tc.input, result, tc.expectedOutput)
			}
		})
	}
}

// TestNewReaderContainerErrors tests more container-related error conditions
func TestNewReaderContainerErrors(t *testing.T) {
	testCases := []struct {
		name           string
		config         LogSourceConfig
		expectedErrMsg string
	}{
		{
			name: "Missing pod name",
			config: LogSourceConfig{
				Type:          ContainerSourceType,
				Namespace:     "default",
				ContainerName: "container",
			},
			expectedErrMsg: "pod name is required",
		},
		{
			name: "Missing container name",
			config: LogSourceConfig{
				Type:      ContainerSourceType,
				Namespace: "default",
				PodName:   "pod",
			},
			expectedErrMsg: "container name is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewReader(tc.config)
			if err == nil {
				t.Fatalf("Expected error but got nil")
			}
			if !strings.Contains(err.Error(), tc.expectedErrMsg) {
				t.Errorf("Error %q does not contain expected message %q", err.Error(), tc.expectedErrMsg)
			}
		})
	}
}

// TestWindowsEventLogDefaultLevel tests default level for Windows Event Log
func TestWindowsEventLogDefaultLevel(t *testing.T) {
	// Skip on non-Windows platforms
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows platform")
	}

	// Store the original factory function
	originalFactory := windowsEventLogReaderFactory
	defer func() { windowsEventLogReaderFactory = originalFactory }()

	// Setup a test factory to capture arguments
	var capturedLogName, capturedLevel string
	windowsEventLogReaderFactory = func(logName, minLevel string) (LogReader, error) {
		capturedLogName = logName
		capturedLevel = minLevel
		return &mockContainerReader{lines: make(chan string)}, nil
	}

	// Test with default level (not specified)
	config := LogSourceConfig{
		Type:                WindowsEventSourceType,
		WindowsEventLogName: "System",
	}
	_, err := NewReader(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify default level was used
	if capturedLogName != "System" {
		t.Errorf("Expected log name 'System', got %q", capturedLogName)
	}
	// The default level depends on implementation, but it shouldn't be empty
	if capturedLevel == "" {
		t.Errorf("Expected a default level, got empty string")
	}
}

// TestMacOSLogQuery tests query handling for macOS log reader
func TestMacOSLogQuery(t *testing.T) {
	// Skip on non-macOS platforms
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test on non-macOS platform")
	}

	// Store the original factory function
	originalFactory := macosLogReaderFactory
	defer func() { macosLogReaderFactory = originalFactory }()

	// Setup a test factory to capture arguments
	var capturedQuery string
	macosLogReaderFactory = func(query string) (LogReader, error) {
		capturedQuery = query
		return &mockContainerReader{lines: make(chan string)}, nil
	}

	// Test with a specific query
	expectedQuery := "process == \"kernel\" AND level == \"error\""
	config := LogSourceConfig{
		Type:          MacOSASLSourceType,
		MacOSLogQuery: expectedQuery,
	}
	_, err := NewReader(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify query was passed correctly
	if capturedQuery != expectedQuery {
		t.Errorf("Expected query %q, got %q", expectedQuery, capturedQuery)
	}

	// Test with empty query
	config = LogSourceConfig{
		Type: MacOSASLSourceType,
	}
	_, err = NewReader(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify empty query was handled correctly
	if capturedQuery != "" {
		t.Errorf("Expected empty query, got %q", capturedQuery)
	}
}

// TestNewReaderWithCustomType tests handling of custom or unsupported source types
func TestNewReaderWithCustomType(t *testing.T) {
	testCases := []struct {
		name           string
		sourceType     LogSourceType
		expectedErrMsg string
	}{
		{
			name:           "Empty source type",
			sourceType:     "",
			expectedErrMsg: "unknown log source type:",
		},
		{
			name:           "Custom source type",
			sourceType:     "custom_type",
			expectedErrMsg: "unknown log source type:",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := LogSourceConfig{
				Type: tc.sourceType,
			}
			_, err := NewReader(config)
			if err == nil {
				t.Fatalf("Expected error for source type %q but got nil", tc.sourceType)
			}
			if !strings.Contains(err.Error(), tc.expectedErrMsg) {
				t.Errorf("Error %q does not contain expected message %q", err.Error(), tc.expectedErrMsg)
			}
		})
	}
}
