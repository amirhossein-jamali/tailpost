package http

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/amirhossein-jamali/tailpost/pkg/config"
	"github.com/amirhossein-jamali/tailpost/pkg/security"
)

// HealthServer provides health endpoints for Kubernetes probes
type HealthServer struct {
	listenAddr   string
	server       *http.Server
	ready        bool
	lock         sync.RWMutex
	authProvider security.AuthProvider
	useTLS       bool
	certFile     string
	keyFile      string
}

// HealthStatus represents the status response
type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Version   string            `json:"version"`
	Info      map[string]string `json:"info,omitempty"`
}

// NewHealthServer creates a new health server
func NewHealthServer(listenAddr string) *HealthServer {
	return &HealthServer{
		listenAddr: listenAddr,
		ready:      false,
	}
}

// NewSecureHealthServer creates a new health server with security features
func NewSecureHealthServer(listenAddr string, securityConfig config.SecurityConfig) (*HealthServer, error) {
	server := &HealthServer{
		listenAddr: listenAddr,
		ready:      false,
	}

	// Set up TLS if enabled
	if securityConfig.TLS.Enabled {
		server.useTLS = true
		server.certFile = securityConfig.TLS.CertFile
		server.keyFile = securityConfig.TLS.KeyFile
	}

	// Set up authentication if enabled
	if securityConfig.Auth.Type != "none" {
		authProvider, err := security.NewAuthProvider(securityConfig.Auth)
		if err != nil {
			return nil, err
		}
		server.authProvider = authProvider
	}

	return server, nil
}

// withAuth wraps a handler with authentication if enabled
func (s *HealthServer) withAuth(handler http.HandlerFunc) http.HandlerFunc {
	if s.authProvider == nil {
		// No authentication, return handler as is
		return handler
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Check authentication
		authenticated, err := s.authProvider.Authenticate(r)
		if err != nil {
			log.Printf("Authentication error: %v", err)
			http.Error(w, "Authentication error", http.StatusInternalServerError)
			return
		}

		if !authenticated {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Authentication successful, call the original handler
		handler(w, r)
	}
}

// Start starts the health server
func (s *HealthServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.withAuth(s.healthHandler))
	mux.HandleFunc("/ready", s.withAuth(s.readyHandler))
	mux.HandleFunc("/metrics", s.withAuth(s.metricsHandler))

	s.server = &http.Server{
		Addr:    s.listenAddr,
		Handler: mux,
	}

	go func() {
		var err error
		if s.useTLS {
			log.Printf("Starting secure health server on https://%s", s.listenAddr)
			err = s.server.ListenAndServeTLS(s.certFile, s.keyFile)
		} else {
			log.Printf("Starting health server on http://%s", s.listenAddr)
			err = s.server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			log.Printf("Health server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the health server
func (s *HealthServer) Stop() error {
	if s.server != nil {
		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Gracefully shut down the server
		if err := s.server.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}

// SetReady sets the ready status
func (s *HealthServer) SetReady(ready bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.ready = ready
}

// IsReady returns the ready status
func (s *HealthServer) IsReady() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.ready
}

// SetTLSConfig sets a custom TLS configuration
func (s *HealthServer) SetTLSConfig(tlsConfig *tls.Config) {
	if s.server != nil && tlsConfig != nil {
		s.server.TLSConfig = tlsConfig
	}
}

// healthHandler handles health checks
func (s *HealthServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	status := HealthStatus{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// readyHandler handles readiness checks
func (s *HealthServer) readyHandler(w http.ResponseWriter, r *http.Request) {
	if s.IsReady() {
		status := HealthStatus{
			Status:    "ready",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   "1.0.0",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(status)
	} else {
		status := HealthStatus{
			Status:    "not ready",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   "1.0.0",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(status)
	}
}

// metricsHandler handles metrics requests
func (s *HealthServer) metricsHandler(w http.ResponseWriter, r *http.Request) {
	// Will be replaced with real metrics in a future implementation
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("# HELP tailpost_up Whether the Tailpost agent is running\n"))
	w.Write([]byte("# TYPE tailpost_up gauge\n"))
	w.Write([]byte("tailpost_up 1\n"))
}
