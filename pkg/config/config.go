package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// LogSourceType defines the type of log source
type LogSourceType string

const (
	// FileLogSource represents a file log source
	FileLogSource LogSourceType = "file"
	// ContainerLogSource represents a container log source
	ContainerLogSource LogSourceType = "container"
	// PodLogSource represents a pod log source
	PodLogSource LogSourceType = "pod"
	// WindowsEventLogSource represents a Windows Event Log source
	WindowsEventLogSource LogSourceType = "windows_event"
	// MacOSASLLogSource represents a macOS ASL log source
	MacOSASLLogSource LogSourceType = "macos_asl"
)

// TLSConfig represents TLS configuration for secure communications
type TLSConfig struct {
	Enabled                  bool   `yaml:"enabled"`
	CertFile                 string `yaml:"cert_file"`
	KeyFile                  string `yaml:"key_file"`
	CAFile                   string `yaml:"ca_file"`
	InsecureSkipVerify       bool   `yaml:"insecure_skip_verify"`
	ServerName               string `yaml:"server_name"`
	MinVersion               string `yaml:"min_version"`
	MaxVersion               string `yaml:"max_version"`
	PreferServerCipherSuites bool   `yaml:"prefer_server_cipher_suites"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Type         string            `yaml:"type"`          // none, basic, token, oauth2
	Username     string            `yaml:"username"`      // for basic auth
	Password     string            `yaml:"password"`      // for basic auth
	TokenFile    string            `yaml:"token_file"`    // for token auth
	ClientID     string            `yaml:"client_id"`     // for oauth2
	ClientSecret string            `yaml:"client_secret"` // for oauth2
	TokenURL     string            `yaml:"token_url"`     // for oauth2
	Scopes       []string          `yaml:"scopes"`        // for oauth2
	Headers      map[string]string `yaml:"headers"`       // for custom header auth
}

// EncryptionConfig represents data encryption configuration
type EncryptionConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Type         string `yaml:"type"`          // aes, chacha20poly1305
	Algorithm    string `yaml:"algorithm"`     // algorithm name (for backward compatibility)
	KeyFile      string `yaml:"key_file"`      // path to encryption key file
	KeyEnv       string `yaml:"key_env"`       // environment variable containing encryption key
	KeyID        string `yaml:"key_id"`        // key identifier for rotation
	RotationDays int    `yaml:"rotation_days"` // number of days before key rotation
}

// SecurityConfig represents the security configuration
type SecurityConfig struct {
	TLS        TLSConfig        `yaml:"tls"`
	Auth       AuthConfig       `yaml:"auth"`
	Encryption EncryptionConfig `yaml:"encryption"`
}

// TelemetryConfig represents the configuration for telemetry
type TelemetryConfig struct {
	Enabled            bool              `yaml:"enabled"`
	ServiceName        string            `yaml:"service_name"`
	ServiceVersion     string            `yaml:"service_version"`
	ExporterType       string            `yaml:"exporter_type"`
	ExporterEndpoint   string            `yaml:"exporter_endpoint"`
	SamplingRate       float64           `yaml:"sampling_rate"`
	ContextPropagation bool              `yaml:"context_propagation"`
	Attributes         map[string]string `yaml:"attributes"`
}

// Config represents the configuration for the application
type Config struct {
	// Common fields
	LogPath       string        `yaml:"log_path"`
	ServerURL     string        `yaml:"server_url"`
	BatchSize     int           `yaml:"batch_size"`
	FlushInterval time.Duration `yaml:"flush_interval"`

	// Kubernetes fields
	LogSourceType     LogSourceType     `yaml:"log_source_type"`
	Namespace         string            `yaml:"namespace"`
	PodName           string            `yaml:"pod_name"`
	ContainerName     string            `yaml:"container_name"`
	PodSelector       map[string]string `yaml:"pod_selector"`
	NamespaceSelector map[string]string `yaml:"namespace_selector"`

	// Windows Event Log fields
	WindowsEventLogName  string `yaml:"windows_event_log_name"`
	WindowsEventLogLevel string `yaml:"windows_event_log_level"`

	// macOS ASL fields
	MacOSLogQuery string `yaml:"macos_log_query"`

	// Telemetry configuration
	Telemetry TelemetryConfig `yaml:"telemetry"`

	// Security configuration
	Security SecurityConfig `yaml:"security"`
}

// getDefaultLogPath returns the default log path based on OS
func getDefaultLogPath() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("SYSTEMROOT"), "Logs", "Application.log")
	case "darwin": // macOS
		return "/var/log/system.log"
	default: // Linux and others
		return "/var/log/syslog"
	}
}

// getDefaultLogSourceType returns the default log source type based on OS
func getDefaultLogSourceType() LogSourceType {
	switch runtime.GOOS {
	case "windows":
		return WindowsEventLogSource
	case "darwin":
		return MacOSASLLogSource
	default: // Linux and others
		return FileLogSource
	}
}

// GetDefaultLogSourceType returns the default log source type based on OS (exported for tests)
func GetDefaultLogSourceType() LogSourceType {
	return getDefaultLogSourceType()
}

// DefaultTelemetryConfig returns the default telemetry configuration
func DefaultTelemetryConfig() TelemetryConfig {
	return TelemetryConfig{
		Enabled:            false,
		ServiceName:        "tailpost",
		ServiceVersion:     "0.1.0",
		ExporterType:       "http",
		ExporterEndpoint:   "http://localhost:4318",
		SamplingRate:       1.0,
		ContextPropagation: true,
		Attributes:         map[string]string{},
	}
}

// DefaultSecurityConfig returns the default security configuration
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		TLS: TLSConfig{
			Enabled:                  false,
			InsecureSkipVerify:       false,
			MinVersion:               "tls12",
			PreferServerCipherSuites: true,
		},
		Auth: AuthConfig{
			Type: "none",
		},
		Encryption: EncryptionConfig{
			Enabled:      false,
			Type:         "aes",
			RotationDays: 90,
		},
	}
}

// LoadConfig loads the configuration from a YAML file
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	// Set defaults if not provided
	if config.BatchSize == 0 {
		config.BatchSize = 10
	}
	if config.FlushInterval == 0 {
		config.FlushInterval = 5 * time.Second
	}

	// Set OS-specific defaults for log source type if not specified
	if config.LogSourceType == "" {
		config.LogSourceType = getDefaultLogSourceType()
	}

	// Set platform-specific defaults
	if config.LogSourceType == WindowsEventLogSource && config.WindowsEventLogName == "" {
		config.WindowsEventLogName = "Application"
	}

	if config.LogSourceType == WindowsEventLogSource && config.WindowsEventLogLevel == "" {
		config.WindowsEventLogLevel = "Information"
	}

	// Set default telemetry configuration
	defaultTelemetry := DefaultTelemetryConfig()
	// For telemetry, always ensure we have defaults in place, even if some fields are custom
	if config.Telemetry.Enabled {
		// Only override with defaults for unspecified fields when telemetry is enabled
		if config.Telemetry.ServiceName == "" {
			config.Telemetry.ServiceName = defaultTelemetry.ServiceName
		}
		if config.Telemetry.ServiceVersion == "" {
			config.Telemetry.ServiceVersion = defaultTelemetry.ServiceVersion
		}
		if config.Telemetry.ExporterType == "" {
			config.Telemetry.ExporterType = defaultTelemetry.ExporterType
		}
		if config.Telemetry.ExporterEndpoint == "" {
			config.Telemetry.ExporterEndpoint = defaultTelemetry.ExporterEndpoint
		}
		if config.Telemetry.SamplingRate == 0 {
			config.Telemetry.SamplingRate = defaultTelemetry.SamplingRate
		}
		if config.Telemetry.Attributes == nil {
			config.Telemetry.Attributes = defaultTelemetry.Attributes
		}
	} else {
		// If telemetry is not enabled, just use all defaults
		config.Telemetry = defaultTelemetry
	}

	// Set default security configuration
	defaultSecurity := DefaultSecurityConfig()
	// Apply defaults for security settings
	if config.Security.TLS.Enabled {
		// Only set defaults for TLS fields that aren't specified
		if config.Security.TLS.MinVersion == "" {
			config.Security.TLS.MinVersion = defaultSecurity.TLS.MinVersion
		}
	} else {
		config.Security.TLS = defaultSecurity.TLS
	}

	if config.Security.Auth.Type == "" {
		config.Security.Auth.Type = defaultSecurity.Auth.Type
	}

	if config.Security.Encryption.Enabled {
		if config.Security.Encryption.Type == "" {
			config.Security.Encryption.Type = defaultSecurity.Encryption.Type
		}
		if config.Security.Encryption.RotationDays == 0 {
			config.Security.Encryption.RotationDays = defaultSecurity.Encryption.RotationDays
		}
	} else {
		config.Security.Encryption = defaultSecurity.Encryption
	}

	// Handle log path with OS detection for file type sources
	if config.LogSourceType == FileLogSource {
		if config.LogPath == "" {
			config.LogPath = getDefaultLogPath()
		} else if strings.HasPrefix(config.LogPath, "${OS_DEFAULT}") {
			// Allow specifying OS-specific suffix like "${OS_DEFAULT}/myapp.log"
			suffix := strings.TrimPrefix(config.LogPath, "${OS_DEFAULT}")
			basePath := filepath.Dir(getDefaultLogPath())
			config.LogPath = filepath.Join(basePath, suffix)
		}
	}

	// Validate required fields based on source type
	if config.LogSourceType == FileLogSource {
		if config.LogPath == "" {
			return nil, fmt.Errorf("log_path is required for file log source")
		}
	} else if config.LogSourceType == ContainerLogSource {
		if config.Namespace == "" {
			return nil, fmt.Errorf("namespace is required for container log source")
		}
		if config.PodName == "" {
			return nil, fmt.Errorf("pod_name is required for container log source")
		}
		if config.ContainerName == "" {
			return nil, fmt.Errorf("container_name is required for container log source")
		}
	} else if config.LogSourceType == PodLogSource {
		if len(config.PodSelector) == 0 {
			return nil, fmt.Errorf("pod_selector is required for pod log source")
		}
	} else if config.LogSourceType == WindowsEventLogSource {
		if runtime.GOOS != "windows" {
			return nil, fmt.Errorf("windows_event log source type is only supported on Windows")
		}
	} else if config.LogSourceType == MacOSASLLogSource {
		if runtime.GOOS != "darwin" {
			return nil, fmt.Errorf("macos_asl log source type is only supported on macOS")
		}
	}

	// Validate security configuration if enabled
	if config.Security.TLS.Enabled {
		// Validate TLS configuration
		if config.Security.TLS.CertFile == "" && config.ServerURL != "" && strings.HasPrefix(config.ServerURL, "https://") {
			return nil, fmt.Errorf("cert_file is required when TLS is enabled for HTTPS connections")
		}
		if config.Security.TLS.KeyFile == "" && config.Security.TLS.CertFile != "" {
			return nil, fmt.Errorf("key_file is required when cert_file is specified")
		}
	}

	if config.Security.Auth.Type != "none" {
		// Validate auth configuration based on type
		switch config.Security.Auth.Type {
		case "basic":
			if config.Security.Auth.Username == "" || config.Security.Auth.Password == "" {
				return nil, fmt.Errorf("username and password are required for basic authentication")
			}
		case "token":
			if config.Security.Auth.TokenFile == "" {
				return nil, fmt.Errorf("token_file is required for token authentication")
			}
		case "oauth2":
			if config.Security.Auth.ClientID == "" || config.Security.Auth.ClientSecret == "" || config.Security.Auth.TokenURL == "" {
				return nil, fmt.Errorf("client_id, client_secret, and token_url are required for OAuth2 authentication")
			}
		}
	}

	if config.Security.Encryption.Enabled {
		// Validate encryption configuration
		if config.Security.Encryption.KeyFile == "" && config.Security.Encryption.KeyEnv == "" {
			return nil, fmt.Errorf("either key_file or key_env must be specified when encryption is enabled")
		}
	}

	// Always validate server_url
	if config.ServerURL == "" {
		return nil, fmt.Errorf("server_url is required in config")
	}

	return &config, nil
}
