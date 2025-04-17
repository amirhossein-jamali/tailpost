package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tempFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test data
	configContent := `
log_source_type: file
log_path: /var/log/test.log
server_url: http://example.com/logs
batch_size: 20
flush_interval: 10s
`
	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify values
	if cfg.LogSourceType != FileLogSource {
		t.Errorf("Expected log_source_type to be 'file', got '%s'", cfg.LogSourceType)
	}
	if cfg.LogPath != "/var/log/test.log" {
		t.Errorf("Expected log_path to be '/var/log/test.log', got '%s'", cfg.LogPath)
	}
	if cfg.ServerURL != "http://example.com/logs" {
		t.Errorf("Expected server_url to be 'http://example.com/logs', got '%s'", cfg.ServerURL)
	}
	if cfg.BatchSize != 20 {
		t.Errorf("Expected batch_size to be 20, got %d", cfg.BatchSize)
	}
	if cfg.FlushInterval != 10*time.Second {
		t.Errorf("Expected flush_interval to be 10s, got %s", cfg.FlushInterval)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	// Create a temporary config file with minimal settings
	tempFile, err := os.CreateTemp("", "config-defaults-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write minimal config data
	configContent := `
log_path: /var/log/minimal.log
server_url: http://minimal.example.com/logs
`
	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify default values are set
	if cfg.BatchSize != 10 {
		t.Errorf("Expected default BatchSize to be 10, got %d", cfg.BatchSize)
	}
	if cfg.FlushInterval != 5*time.Second {
		t.Errorf("Expected default FlushInterval to be 5s, got %s", cfg.FlushInterval)
	}

	// Verify the log source type is set to the OS-specific default
	expectedSourceType := getDefaultLogSourceType()
	if cfg.LogSourceType != expectedSourceType {
		t.Errorf("Expected default LogSourceType to be %s, got %s", expectedSourceType, cfg.LogSourceType)
	}
}

func TestLoadConfigWindowsEventLog(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows event log test on non-Windows platform")
	}

	// Create a temporary config file for Windows Event Log
	tempFile, err := os.CreateTemp("", "config-windows-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write Windows Event Log config
	configContent := `
log_source_type: windows_event
windows_event_log_name: System
windows_event_log_level: Warning
server_url: http://example.com/logs
`
	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify values
	if cfg.LogSourceType != WindowsEventLogSource {
		t.Errorf("Expected log_source_type to be 'windows_event', got '%s'", cfg.LogSourceType)
	}
	if cfg.WindowsEventLogName != "System" {
		t.Errorf("Expected windows_event_log_name to be 'System', got '%s'", cfg.WindowsEventLogName)
	}
	if cfg.WindowsEventLogLevel != "Warning" {
		t.Errorf("Expected windows_event_log_level to be 'Warning', got '%s'", cfg.WindowsEventLogLevel)
	}
}

func TestLoadConfigMacOSLog(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS log test on non-macOS platform")
	}

	// Create a temporary config file for macOS logs
	tempFile, err := os.CreateTemp("", "config-macos-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write macOS log config
	configContent := `
log_source_type: macos_asl
macos_log_query: "process == \"kernel\""
server_url: http://example.com/logs
`
	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify values
	if cfg.LogSourceType != MacOSASLLogSource {
		t.Errorf("Expected log_source_type to be 'macos_asl', got '%s'", cfg.LogSourceType)
	}
	if cfg.MacOSLogQuery != "process == \"kernel\"" {
		t.Errorf("Expected macos_log_query to be 'process == \"kernel\"', got '%s'", cfg.MacOSLogQuery)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		skipOnOS string
	}{
		{
			name: "Missing server_url",
			content: `
log_source_type: file
log_path: /var/log/test.log
`,
		},
		{
			name: "Missing namespace for container source",
			content: `
log_source_type: container
pod_name: test-pod
container_name: test-container
server_url: http://example.com/logs
`,
		},
		{
			name: "Missing pod_name for container source",
			content: `
log_source_type: container
namespace: default
container_name: test-container
server_url: http://example.com/logs
`,
		},
		{
			name: "Missing container_name for container source",
			content: `
log_source_type: container
namespace: default
pod_name: test-pod
server_url: http://example.com/logs
`,
		},
		{
			name: "Missing pod_selector for pod source",
			content: `
log_source_type: pod
namespace: default
server_url: http://example.com/logs
`,
		},
		{
			name: "Windows event log source on non-Windows",
			content: `
log_source_type: windows_event
windows_event_log_name: Application
server_url: http://example.com/logs
`,
			skipOnOS: "windows",
		},
		{
			name: "macOS log source on non-macOS",
			content: `
log_source_type: macos_asl
server_url: http://example.com/logs
`,
			skipOnOS: "darwin",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Skip tests that don't apply to the current OS
			if tc.skipOnOS != "" && runtime.GOOS == tc.skipOnOS {
				t.Skip("Skipping test on " + tc.skipOnOS)
			}

			// Create temp file for this test case
			tempFile, err := os.CreateTemp("", "config-error-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tempFile.Name())

			// Write config content
			if _, err := tempFile.Write([]byte(tc.content)); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			if err := tempFile.Close(); err != nil {
				t.Fatalf("Failed to close temp file: %v", err)
			}

			// Test loading the config, should error
			_, err = LoadConfig(tempFile.Name())
			if err == nil {
				t.Errorf("Expected error but got nil for test case: %s", tc.name)
			}
		})
	}
}

func TestGetDefaultLogPath(t *testing.T) {
	path := getDefaultLogPath()

	// Check that the path is not empty
	if path == "" {
		t.Errorf("Expected non-empty default log path")
	}

	// Check that the path is absolute
	if !filepath.IsAbs(path) {
		t.Errorf("Expected absolute path, got: %s", path)
	}

	// Check OS-specific paths
	switch runtime.GOOS {
	case "windows":
		if !isPathPrefixOf(filepath.ToSlash(os.Getenv("SYSTEMROOT")), filepath.ToSlash(path)) {
			t.Errorf("Expected Windows path to start with %%SYSTEMROOT%%, got: %s", path)
		}
	case "darwin":
		expected := "/var/log/system.log"
		if path != expected {
			t.Errorf("Expected macOS path to be %s, got: %s", expected, path)
		}
	default: // Linux and others
		expected := "/var/log/syslog"
		if path != expected {
			t.Errorf("Expected Linux path to be %s, got: %s", expected, path)
		}
	}
}

func TestGetDefaultLogSourceType(t *testing.T) {
	sourceType := getDefaultLogSourceType()

	// Check OS-specific types
	switch runtime.GOOS {
	case "windows":
		if sourceType != WindowsEventLogSource {
			t.Errorf("Expected Windows default source type to be WindowsEventLogSource, got: %s", sourceType)
		}
	case "darwin":
		if sourceType != MacOSASLLogSource {
			t.Errorf("Expected macOS default source type to be MacOSASLLogSource, got: %s", sourceType)
		}
	default: // Linux and others
		if sourceType != FileLogSource {
			t.Errorf("Expected Linux default source type to be FileLogSource, got: %s", sourceType)
		}
	}
}

func TestLoadConfigWithOSDefaultPath(t *testing.T) {
	// Create a temporary config file
	tempFile, err := os.CreateTemp("", "config-os-default-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test data with ${OS_DEFAULT} placeholder
	configContent := `
log_source_type: file
log_path: ${OS_DEFAULT}/myapp.log
server_url: http://example.com/logs
`
	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Get expected base directory based on OS
	basePath := filepath.Dir(getDefaultLogPath())
	expected := filepath.Join(basePath, "myapp.log")

	// Verify the log path was set correctly
	if cfg.LogPath != expected {
		t.Errorf("Expected log_path to be '%s', got '%s'", expected, cfg.LogPath)
	}
}

func TestFileSourceDefaultLogPath(t *testing.T) {
	// Create a temporary config file
	tempFile, err := os.CreateTemp("", "config-default-path-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test data without log_path
	configContent := `
log_source_type: file
server_url: http://example.com/logs
batch_size: 20
flush_interval: 10s
`
	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify that log_path got the default value
	expectedPath := getDefaultLogPath()
	if cfg.LogPath != expectedPath {
		t.Errorf("Expected default log_path to be '%s', got '%s'", expectedPath, cfg.LogPath)
	}
}

// NEW TESTS BELOW

// Test for DefaultTelemetryConfig function
func TestDefaultTelemetryConfig(t *testing.T) {
	defaultConfig := DefaultTelemetryConfig()

	// Verify all default values
	if defaultConfig.Enabled != false {
		t.Errorf("Expected default telemetry enabled to be false, got %v", defaultConfig.Enabled)
	}
	if defaultConfig.ServiceName != "tailpost" {
		t.Errorf("Expected default service name to be 'tailpost', got '%s'", defaultConfig.ServiceName)
	}
	if defaultConfig.ServiceVersion != "0.1.0" {
		t.Errorf("Expected default service version to be '0.1.0', got '%s'", defaultConfig.ServiceVersion)
	}
	if defaultConfig.ExporterType != "http" {
		t.Errorf("Expected default exporter type to be 'http', got '%s'", defaultConfig.ExporterType)
	}
	if defaultConfig.ExporterEndpoint != "http://localhost:4318" {
		t.Errorf("Expected default exporter endpoint to be 'http://localhost:4318', got '%s'", defaultConfig.ExporterEndpoint)
	}
	if defaultConfig.SamplingRate != 1.0 {
		t.Errorf("Expected default sampling rate to be 1.0, got %v", defaultConfig.SamplingRate)
	}
	if defaultConfig.ContextPropagation != true {
		t.Errorf("Expected default context propagation to be true, got %v", defaultConfig.ContextPropagation)
	}
	if defaultConfig.Attributes == nil {
		t.Errorf("Expected default attributes to be non-nil map")
	}
}

// Test for loading config with telemetry settings
func TestLoadConfigWithTelemetry(t *testing.T) {
	// Create a temporary config file
	tempFile, err := os.CreateTemp("", "config-telemetry-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test data with telemetry configuration
	configContent := `
log_source_type: file
log_path: /var/log/test.log
server_url: http://example.com/logs
telemetry:
  enabled: true
  service_name: "test-service"
  service_version: "1.0.0"
  exporter_type: "grpc"
  exporter_endpoint: "localhost:4317"
  sampling_rate: 0.5
  context_propagation: false
  attributes:
    env: "test"
    region: "us-west"
`
	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify telemetry values
	if !cfg.Telemetry.Enabled {
		t.Errorf("Expected telemetry.enabled to be true")
	}
	if cfg.Telemetry.ServiceName != "test-service" {
		t.Errorf("Expected telemetry.service_name to be 'test-service', got '%s'", cfg.Telemetry.ServiceName)
	}
	if cfg.Telemetry.ServiceVersion != "1.0.0" {
		t.Errorf("Expected telemetry.service_version to be '1.0.0', got '%s'", cfg.Telemetry.ServiceVersion)
	}
	if cfg.Telemetry.ExporterType != "grpc" {
		t.Errorf("Expected telemetry.exporter_type to be 'grpc', got '%s'", cfg.Telemetry.ExporterType)
	}
	if cfg.Telemetry.ExporterEndpoint != "localhost:4317" {
		t.Errorf("Expected telemetry.exporter_endpoint to be 'localhost:4317', got '%s'", cfg.Telemetry.ExporterEndpoint)
	}
	if cfg.Telemetry.SamplingRate != 0.5 {
		t.Errorf("Expected telemetry.sampling_rate to be 0.5, got %v", cfg.Telemetry.SamplingRate)
	}
	if cfg.Telemetry.ContextPropagation {
		t.Errorf("Expected telemetry.context_propagation to be false")
	}
	if env, ok := cfg.Telemetry.Attributes["env"]; !ok || env != "test" {
		t.Errorf("Expected telemetry.attributes to have env='test', got %v", cfg.Telemetry.Attributes)
	}
	if region, ok := cfg.Telemetry.Attributes["region"]; !ok || region != "us-west" {
		t.Errorf("Expected telemetry.attributes to have region='us-west', got %v", cfg.Telemetry.Attributes)
	}
}

// Test for loading config with partially specified telemetry
func TestLoadConfigWithPartialTelemetry(t *testing.T) {
	// Create a temporary config file
	tempFile, err := os.CreateTemp("", "config-partial-telemetry-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test data with minimal telemetry configuration
	configContent := `
log_source_type: file
log_path: /var/log/test.log
server_url: http://example.com/logs
telemetry:
  enabled: true
  service_name: "custom-service"
`
	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify custom value is preserved
	if cfg.Telemetry.ServiceName != "custom-service" {
		t.Errorf("Expected telemetry.service_name to be 'custom-service', got '%s'", cfg.Telemetry.ServiceName)
	}

	// Verify default values are set for unspecified fields
	defaultConfig := DefaultTelemetryConfig()
	if cfg.Telemetry.ServiceVersion != defaultConfig.ServiceVersion {
		t.Errorf("Expected telemetry.service_version to be default '%s', got '%s'",
			defaultConfig.ServiceVersion, cfg.Telemetry.ServiceVersion)
	}
	if cfg.Telemetry.ExporterType != defaultConfig.ExporterType {
		t.Errorf("Expected telemetry.exporter_type to be default '%s', got '%s'",
			defaultConfig.ExporterType, cfg.Telemetry.ExporterType)
	}
}

// Test for pod log source configuration
func TestLoadConfigPodLogSource(t *testing.T) {
	// Create a temporary config file
	tempFile, err := os.CreateTemp("", "config-pod-source-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test data with pod log source
	configContent := `
log_source_type: pod
pod_selector:
  app: nginx
  tier: frontend
namespace: kube-system
server_url: http://example.com/logs
`
	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify values
	if cfg.LogSourceType != PodLogSource {
		t.Errorf("Expected log_source_type to be 'pod', got '%s'", cfg.LogSourceType)
	}
	if len(cfg.PodSelector) != 2 {
		t.Errorf("Expected pod_selector to have 2 items, got %d", len(cfg.PodSelector))
	}
	if app, ok := cfg.PodSelector["app"]; !ok || app != "nginx" {
		t.Errorf("Expected pod_selector to have app='nginx', got %v", cfg.PodSelector)
	}
	if tier, ok := cfg.PodSelector["tier"]; !ok || tier != "frontend" {
		t.Errorf("Expected pod_selector to have tier='frontend', got %v", cfg.PodSelector)
	}
	if cfg.Namespace != "kube-system" {
		t.Errorf("Expected namespace to be 'kube-system', got '%s'", cfg.Namespace)
	}
}

// Test for loading configuration with namespace selector
func TestLoadConfigWithNamespaceSelector(t *testing.T) {
	// Create a temporary config file
	tempFile, err := os.CreateTemp("", "config-namespace-selector-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test data with namespace selector
	configContent := `
log_source_type: pod
pod_selector:
  app: backend
namespace_selector:
  environment: production
  region: us-east-1
server_url: http://example.com/logs
`
	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify namespace selector values
	if len(cfg.NamespaceSelector) != 2 {
		t.Errorf("Expected namespace_selector to have 2 items, got %d", len(cfg.NamespaceSelector))
	}
	if env, ok := cfg.NamespaceSelector["environment"]; !ok || env != "production" {
		t.Errorf("Expected namespace_selector to have environment='production', got %v", cfg.NamespaceSelector)
	}
	if region, ok := cfg.NamespaceSelector["region"]; !ok || region != "us-east-1" {
		t.Errorf("Expected namespace_selector to have region='us-east-1', got %v", cfg.NamespaceSelector)
	}
}

// Test for container log source configuration
func TestLoadConfigContainerLogSource(t *testing.T) {
	// Create a temporary config file
	tempFile, err := os.CreateTemp("", "config-container-source-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test data with container log source
	configContent := `
log_source_type: container
namespace: default
pod_name: web-app
container_name: nginx
server_url: http://example.com/logs
`
	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify values
	if cfg.LogSourceType != ContainerLogSource {
		t.Errorf("Expected log_source_type to be 'container', got '%s'", cfg.LogSourceType)
	}
	if cfg.Namespace != "default" {
		t.Errorf("Expected namespace to be 'default', got '%s'", cfg.Namespace)
	}
	if cfg.PodName != "web-app" {
		t.Errorf("Expected pod_name to be 'web-app', got '%s'", cfg.PodName)
	}
	if cfg.ContainerName != "nginx" {
		t.Errorf("Expected container_name to be 'nginx', got '%s'", cfg.ContainerName)
	}
}

// Test for DefaultSecurityConfig function
func TestDefaultSecurityConfig(t *testing.T) {
	defaultConfig := DefaultSecurityConfig()

	// Verify all default values for TLS
	if defaultConfig.TLS.Enabled {
		t.Errorf("Expected default TLS enabled to be false, got %v", defaultConfig.TLS.Enabled)
	}
	if defaultConfig.TLS.InsecureSkipVerify {
		t.Errorf("Expected default InsecureSkipVerify to be false, got %v", defaultConfig.TLS.InsecureSkipVerify)
	}
	if defaultConfig.TLS.MinVersion != "tls12" {
		t.Errorf("Expected default MinVersion to be 'tls12', got '%s'", defaultConfig.TLS.MinVersion)
	}
	if !defaultConfig.TLS.PreferServerCipherSuites {
		t.Errorf("Expected default PreferServerCipherSuites to be true, got %v", defaultConfig.TLS.PreferServerCipherSuites)
	}

	// Verify default Auth values
	if defaultConfig.Auth.Type != "none" {
		t.Errorf("Expected default Auth type to be 'none', got '%s'", defaultConfig.Auth.Type)
	}

	// Verify default Encryption values
	if defaultConfig.Encryption.Enabled {
		t.Errorf("Expected default Encryption enabled to be false, got %v", defaultConfig.Encryption.Enabled)
	}
	if defaultConfig.Encryption.Type != "aes" {
		t.Errorf("Expected default Encryption type to be 'aes', got '%s'", defaultConfig.Encryption.Type)
	}
	if defaultConfig.Encryption.RotationDays != 90 {
		t.Errorf("Expected default RotationDays to be 90, got %d", defaultConfig.Encryption.RotationDays)
	}
}

// Test for loading config with TLS settings
func TestLoadConfigWithTLS(t *testing.T) {
	// Create a temporary config file
	tempFile, err := os.CreateTemp("", "config-tls-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test data with TLS configuration
	configContent := `
log_source_type: file
log_path: /var/log/test.log
server_url: https://example.com/logs
security:
  tls:
    enabled: true
    cert_file: /path/to/cert.crt
    key_file: /path/to/key.key
    ca_file: /path/to/ca.crt
    insecure_skip_verify: false
    server_name: example.com
    min_version: tls12
    max_version: tls13
    prefer_server_cipher_suites: true
`
	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify TLS values
	if !cfg.Security.TLS.Enabled {
		t.Errorf("Expected security.tls.enabled to be true")
	}
	if cfg.Security.TLS.CertFile != "/path/to/cert.crt" {
		t.Errorf("Expected security.tls.cert_file to be '/path/to/cert.crt', got '%s'", cfg.Security.TLS.CertFile)
	}
	if cfg.Security.TLS.KeyFile != "/path/to/key.key" {
		t.Errorf("Expected security.tls.key_file to be '/path/to/key.key', got '%s'", cfg.Security.TLS.KeyFile)
	}
	if cfg.Security.TLS.CAFile != "/path/to/ca.crt" {
		t.Errorf("Expected security.tls.ca_file to be '/path/to/ca.crt', got '%s'", cfg.Security.TLS.CAFile)
	}
	if cfg.Security.TLS.InsecureSkipVerify {
		t.Errorf("Expected security.tls.insecure_skip_verify to be false")
	}
	if cfg.Security.TLS.ServerName != "example.com" {
		t.Errorf("Expected security.tls.server_name to be 'example.com', got '%s'", cfg.Security.TLS.ServerName)
	}
	if cfg.Security.TLS.MinVersion != "tls12" {
		t.Errorf("Expected security.tls.min_version to be 'tls12', got '%s'", cfg.Security.TLS.MinVersion)
	}
	if cfg.Security.TLS.MaxVersion != "tls13" {
		t.Errorf("Expected security.tls.max_version to be 'tls13', got '%s'", cfg.Security.TLS.MaxVersion)
	}
	if !cfg.Security.TLS.PreferServerCipherSuites {
		t.Errorf("Expected security.tls.prefer_server_cipher_suites to be true")
	}
}

// Test for loading config with authentication settings
func TestLoadConfigWithAuth(t *testing.T) {
	// Create a temporary config file
	tempFile, err := os.CreateTemp("", "config-auth-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test data with authentication configuration
	configContent := `
log_source_type: file
log_path: /var/log/test.log
server_url: http://example.com/logs
security:
  auth:
    type: basic
    username: testuser
    password: testpass
`
	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify auth values
	if cfg.Security.Auth.Type != "basic" {
		t.Errorf("Expected security.auth.type to be 'basic', got '%s'", cfg.Security.Auth.Type)
	}
	if cfg.Security.Auth.Username != "testuser" {
		t.Errorf("Expected security.auth.username to be 'testuser', got '%s'", cfg.Security.Auth.Username)
	}
	if cfg.Security.Auth.Password != "testpass" {
		t.Errorf("Expected security.auth.password to be 'testpass', got '%s'", cfg.Security.Auth.Password)
	}
}

// Test for loading config with encryption settings
func TestLoadConfigWithEncryption(t *testing.T) {
	// Create a temporary config file
	tempFile, err := os.CreateTemp("", "config-encryption-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test data with encryption configuration
	configContent := `
log_source_type: file
log_path: /var/log/test.log
server_url: http://example.com/logs
security:
  encryption:
    enabled: true
    type: chacha20poly1305
    key_file: /path/to/key.bin
    key_id: test-key-2023
    rotation_days: 30
`
	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify encryption values
	if !cfg.Security.Encryption.Enabled {
		t.Errorf("Expected security.encryption.enabled to be true")
	}
	if cfg.Security.Encryption.Type != "chacha20poly1305" {
		t.Errorf("Expected security.encryption.type to be 'chacha20poly1305', got '%s'", cfg.Security.Encryption.Type)
	}
	if cfg.Security.Encryption.KeyFile != "/path/to/key.bin" {
		t.Errorf("Expected security.encryption.key_file to be '/path/to/key.bin', got '%s'", cfg.Security.Encryption.KeyFile)
	}
	if cfg.Security.Encryption.KeyID != "test-key-2023" {
		t.Errorf("Expected security.encryption.key_id to be 'test-key-2023', got '%s'", cfg.Security.Encryption.KeyID)
	}
	if cfg.Security.Encryption.RotationDays != 30 {
		t.Errorf("Expected security.encryption.rotation_days to be 30, got %d", cfg.Security.Encryption.RotationDays)
	}
}

// Test validation errors for security configuration
func TestLoadConfigSecurityValidation(t *testing.T) {
	testCases := []struct {
		name    string
		content string
	}{
		{
			name: "Missing cert file for HTTPS with TLS",
			content: `
log_source_type: file
log_path: /var/log/test.log
server_url: https://example.com/logs
security:
  tls:
    enabled: true
`,
		},
		{
			name: "Missing key file with cert file",
			content: `
log_source_type: file
log_path: /var/log/test.log
server_url: http://example.com/logs
security:
  tls:
    enabled: true
    cert_file: /path/to/cert.crt
`,
		},
		{
			name: "Basic auth missing username",
			content: `
log_source_type: file
log_path: /var/log/test.log
server_url: http://example.com/logs
security:
  auth:
    type: basic
    password: testpass
`,
		},
		{
			name: "Token auth missing token file",
			content: `
log_source_type: file
log_path: /var/log/test.log
server_url: http://example.com/logs
security:
  auth:
    type: token
`,
		},
		{
			name: "OAuth2 missing client ID",
			content: `
log_source_type: file
log_path: /var/log/test.log
server_url: http://example.com/logs
security:
  auth:
    type: oauth2
    client_secret: secret
    token_url: http://auth.example.com/token
`,
		},
		{
			name: "Encryption missing key sources",
			content: `
log_source_type: file
log_path: /var/log/test.log
server_url: http://example.com/logs
security:
  encryption:
    enabled: true
    type: aes
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temp file for this test case
			tempFile, err := os.CreateTemp("", "config-security-error-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tempFile.Name())

			// Write config content
			if _, err := tempFile.Write([]byte(tc.content)); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			if err := tempFile.Close(); err != nil {
				t.Fatalf("Failed to close temp file: %v", err)
			}

			// Test loading the config, should error
			_, err = LoadConfig(tempFile.Name())
			if err == nil {
				t.Errorf("Expected error but got nil for test case: %s", tc.name)
			}
		})
	}
}

// Test for loading config with partial security settings
func TestLoadConfigWithPartialSecurity(t *testing.T) {
	// Create a temporary config file
	tempFile, err := os.CreateTemp("", "config-partial-security-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test data with minimal security configuration
	configContent := `
log_source_type: file
log_path: /var/log/test.log
server_url: http://example.com/logs
security:
  tls:
    enabled: true
    cert_file: /path/to/cert.crt
    key_file: /path/to/key.key
`
	if _, err := tempFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify TLS settings are preserved
	if !cfg.Security.TLS.Enabled {
		t.Errorf("Expected security.tls.enabled to be true")
	}
	if cfg.Security.TLS.CertFile != "/path/to/cert.crt" {
		t.Errorf("Expected security.tls.cert_file to be '/path/to/cert.crt', got '%s'", cfg.Security.TLS.CertFile)
	}

	// Verify default values for unspecified fields
	defaultConfig := DefaultSecurityConfig()

	// TLS defaults for unspecified fields
	if cfg.Security.TLS.MinVersion != defaultConfig.TLS.MinVersion {
		t.Errorf("Expected security.tls.min_version to be default '%s', got '%s'",
			defaultConfig.TLS.MinVersion, cfg.Security.TLS.MinVersion)
	}

	// Auth should have default values
	if cfg.Security.Auth.Type != defaultConfig.Auth.Type {
		t.Errorf("Expected security.auth.type to be default '%s', got '%s'",
			defaultConfig.Auth.Type, cfg.Security.Auth.Type)
	}

	// Encryption should have default values
	if cfg.Security.Encryption.Enabled != defaultConfig.Encryption.Enabled {
		t.Errorf("Expected security.encryption.enabled to be default %v, got %v",
			defaultConfig.Encryption.Enabled, cfg.Security.Encryption.Enabled)
	}
	if cfg.Security.Encryption.Type != defaultConfig.Encryption.Type {
		t.Errorf("Expected security.encryption.type to be default '%s', got '%s'",
			defaultConfig.Encryption.Type, cfg.Security.Encryption.Type)
	}
}

// Function to use instead of directly using filepath.HasPrefix
func isPathPrefixOf(prefix, path string) bool {
	// Standardize paths by converting to slash
	prefixSlash := filepath.ToSlash(prefix)
	pathSlash := filepath.ToSlash(path)

	// Check if the second path starts with the prefix of the first path
	// and also check if there is a path separator after the prefix
	return strings.HasPrefix(pathSlash, prefixSlash) &&
		(len(pathSlash) == len(prefixSlash) ||
			pathSlash[len(prefixSlash)] == '/')
}
