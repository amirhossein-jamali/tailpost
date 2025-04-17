package security

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/amirhossein-jamali/tailpost/pkg/config"
)

// TLSVersion maps string versions to tls package constants
var TLSVersion = map[string]uint16{
	"tls10": tls.VersionTLS10,
	"tls11": tls.VersionTLS11,
	"tls12": tls.VersionTLS12,
	"tls13": tls.VersionTLS13,
}

// CreateTLSConfig creates a TLS configuration based on the provided config
func CreateTLSConfig(tlsConfig config.TLSConfig) (*tls.Config, error) {
	if !tlsConfig.Enabled {
		return nil, nil
	}

	cfg := &tls.Config{
		InsecureSkipVerify: tlsConfig.InsecureSkipVerify,
		ServerName:         tlsConfig.ServerName,
	}

	// Set minimum TLS version
	if tlsConfig.MinVersion != "" {
		version, ok := TLSVersion[strings.ToLower(tlsConfig.MinVersion)]
		if !ok {
			return nil, fmt.Errorf("unsupported minimum TLS version: %s", tlsConfig.MinVersion)
		}
		cfg.MinVersion = version
	}

	// Set maximum TLS version if specified
	if tlsConfig.MaxVersion != "" {
		version, ok := TLSVersion[strings.ToLower(tlsConfig.MaxVersion)]
		if !ok {
			return nil, fmt.Errorf("unsupported maximum TLS version: %s", tlsConfig.MaxVersion)
		}
		cfg.MaxVersion = version
	}

	// Load CA cert if specified
	if tlsConfig.CAFile != "" {
		caCert, err := os.ReadFile(tlsConfig.CAFile)
		if err != nil {
			return nil, fmt.Errorf("error reading CA file: %v", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("error adding CA certificate to pool")
		}

		cfg.RootCAs = caCertPool
	}

	// Load client cert and key if specified
	if tlsConfig.CertFile != "" && tlsConfig.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(tlsConfig.CertFile, tlsConfig.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("error loading client certificate and key: %v", err)
		}

		cfg.Certificates = []tls.Certificate{cert}
	}

	return cfg, nil
}

// CreateTLSConfigFromEnv creates a TLS configuration from environment variables
func CreateTLSConfigFromEnv() (*tls.Config, error) {
	// Check if TLS is enabled via environment
	tlsEnabled := false
	if tlsEnabledStr := os.Getenv("TAILPOST_TLS_ENABLED"); tlsEnabledStr != "" {
		var err error
		tlsEnabled, err = strconv.ParseBool(tlsEnabledStr)
		if err != nil {
			return nil, fmt.Errorf("invalid TAILPOST_TLS_ENABLED value: %v", err)
		}
	}

	if !tlsEnabled {
		return nil, nil
	}

	tlsConfig := config.TLSConfig{
		Enabled:            tlsEnabled,
		CertFile:           os.Getenv("TAILPOST_TLS_CERT_FILE"),
		KeyFile:            os.Getenv("TAILPOST_TLS_KEY_FILE"),
		CAFile:             os.Getenv("TAILPOST_TLS_CA_FILE"),
		ServerName:         os.Getenv("TAILPOST_TLS_SERVER_NAME"),
		MinVersion:         os.Getenv("TAILPOST_TLS_MIN_VERSION"),
		MaxVersion:         os.Getenv("TAILPOST_TLS_MAX_VERSION"),
		InsecureSkipVerify: os.Getenv("TAILPOST_TLS_INSECURE_SKIP_VERIFY") == "true",
	}

	return CreateTLSConfig(tlsConfig)
}
