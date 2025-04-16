package reader

import (
	"fmt"
	"runtime"
	"strings"
)

// LogReader is the interface that all log readers must implement
type LogReader interface {
	// Start begins the log reading process
	Start() error
	// Lines returns the channel of log lines
	Lines() <-chan string
	// Stop stops the log reader
	Stop()
}

// LogSourceType represents the type of log source
type LogSourceType string

const (
	// FileSourceType is a log source that reads from a file
	FileSourceType LogSourceType = "file"
	// ContainerSourceType is a log source that reads from a container
	ContainerSourceType LogSourceType = "container"
	// PodSourceType is a log source that reads from a pod
	PodSourceType LogSourceType = "pod"
	// WindowsEventSourceType is a log source that reads from Windows Event logs
	WindowsEventSourceType LogSourceType = "windows_event"
	// MacOSASLSourceType is a log source that reads from macOS ASL
	MacOSASLSourceType LogSourceType = "macos_asl"
)

// LogSourceConfig represents configuration for a log source
type LogSourceConfig struct {
	// Type is the type of log source
	Type LogSourceType
	// Path is the path to the log file (for file type)
	Path string
	// Namespace is the Kubernetes namespace (for container/pod types)
	Namespace string
	// PodName is the name of the pod (for container type)
	PodName string
	// ContainerName is the name of the container (for container type)
	ContainerName string
	// PodSelector is a label selector to match pods (for pod type)
	PodSelector string
	// NamespaceSelector is a label selector to match namespaces (for pod type)
	NamespaceSelector string
	// WindowsEventLogName is the name of Windows event log (e.g., Application, System, Security)
	WindowsEventLogName string
	// WindowsEventLogLevel is the minimum level to collect (e.g., Information, Warning, Error)
	WindowsEventLogLevel string
	// MacOSLogQuery is the predicate query for macOS logs
	MacOSLogQuery string
}

// ParseSourceType parses a source type string
func ParseSourceType(sourceType string) (LogSourceType, error) {
	switch strings.ToLower(sourceType) {
	case string(FileSourceType):
		return FileSourceType, nil
	case string(ContainerSourceType):
		return ContainerSourceType, nil
	case string(PodSourceType):
		return PodSourceType, nil
	case string(WindowsEventSourceType), "windows", "event":
		return WindowsEventSourceType, nil
	case string(MacOSASLSourceType), "macos", "asl":
		return MacOSASLSourceType, nil
	default:
		return "", fmt.Errorf("unknown log source type: %s", sourceType)
	}
}

// NewReader creates a new log reader based on the source configuration
func NewReader(config LogSourceConfig) (LogReader, error) {
	switch config.Type {
	case FileSourceType:
		if config.Path == "" {
			return nil, fmt.Errorf("path is required for file source type")
		}
		return NewFileReader(config.Path), nil

	case ContainerSourceType:
		if config.Namespace == "" {
			return nil, fmt.Errorf("namespace is required for container source type")
		}
		if config.PodName == "" {
			return nil, fmt.Errorf("pod name is required for container source type")
		}
		if config.ContainerName == "" {
			return nil, fmt.Errorf("container name is required for container source type")
		}
		return NewContainerReader(config.Namespace, config.PodName, config.ContainerName)

	case PodSourceType:
		return nil, fmt.Errorf("pod source type not implemented yet")

	case WindowsEventSourceType:
		if runtime.GOOS != "windows" {
			return nil, fmt.Errorf("windows event log source type is only supported on Windows")
		}
		if config.WindowsEventLogName == "" {
			config.WindowsEventLogName = "Application" // Default to Application log
		}
		if config.WindowsEventLogLevel == "" {
			config.WindowsEventLogLevel = "Information" // Default to Information level
		}
		return newWindowsEventLogReader(config.WindowsEventLogName, config.WindowsEventLogLevel)

	case MacOSASLSourceType:
		if runtime.GOOS != "darwin" {
			return nil, fmt.Errorf("macOS ASL source type is only supported on macOS")
		}
		return newMacOSLogReader(config.MacOSLogQuery)

	default:
		return nil, fmt.Errorf("unknown log source type: %s", config.Type)
	}
}

// newMacOSLogReader is a platform-agnostic wrapper around the platform-specific implementation
func newMacOSLogReader(query string) (LogReader, error) {
	return macosLogReaderFactory(query)
}

// Default implementation that returns an error for non-macOS platforms
var macosLogReaderFactory = func(query string) (LogReader, error) {
	return nil, fmt.Errorf("macOS log reader is only available on macOS")
}

// newWindowsEventLogReader is a platform-agnostic wrapper around the platform-specific implementation
func newWindowsEventLogReader(logName, minLevel string) (LogReader, error) {
	return windowsEventLogReaderFactory(logName, minLevel)
}

// Default implementation that returns an error for non-Windows platforms
var windowsEventLogReaderFactory = func(logName, minLevel string) (LogReader, error) {
	return nil, fmt.Errorf("windows event log reader is only available on Windows")
}
