package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/amirhossein-jamali/tailpost/pkg/config"
	"github.com/amirhossein-jamali/tailpost/pkg/security"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// HTTPSender represents a component that sends log batches to a server
type HTTPSender struct {
	serverURL          string
	batchSize          int
	flushInterval      time.Duration
	client             *http.Client
	batch              []string
	lock               sync.Mutex
	stopCh             chan struct{}
	stoppedCh          chan struct{}
	tracer             trace.Tracer
	authProvider       security.AuthProvider
	encryptionProvider security.EncryptionProvider
}

// NewHTTPSender creates a new HTTP sender
func NewHTTPSender(serverURL string, batchSize int, flushInterval time.Duration) *HTTPSender {
	// Use default values for invalid parameters
	if batchSize <= 0 {
		batchSize = 1
	}
	if flushInterval <= 0 {
		flushInterval = 1 * time.Second
	}

	return &HTTPSender{
		serverURL:     serverURL,
		batchSize:     batchSize,
		flushInterval: flushInterval,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		batch:     make([]string, 0, batchSize),
		stopCh:    make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}
}

// NewSecureHTTPSender creates a new HTTP sender with security features
func NewSecureHTTPSender(cfg *config.Config) (*HTTPSender, error) {
	sender := &HTTPSender{
		serverURL:     cfg.ServerURL,
		batchSize:     cfg.BatchSize,
		flushInterval: cfg.FlushInterval,
		batch:         make([]string, 0, cfg.BatchSize),
		stopCh:        make(chan struct{}),
		stoppedCh:     make(chan struct{}),
	}

	// Create HTTP client with default timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Configure TLS if enabled
	if cfg.Security.TLS.Enabled {
		tlsConfig, err := security.CreateTLSConfig(cfg.Security.TLS)
		if err != nil {
			return nil, fmt.Errorf("error creating TLS config: %v", err)
		}

		if tlsConfig != nil {
			client.Transport = &http.Transport{
				TLSClientConfig: tlsConfig,
			}
			log.Println("TLS configuration applied to HTTP client")
		}
	}

	sender.client = client

	// Configure authentication if enabled
	if cfg.Security.Auth.Type != "none" {
		authProvider, err := security.NewAuthProvider(cfg.Security.Auth)
		if err != nil {
			return nil, fmt.Errorf("error creating auth provider: %v", err)
		}
		sender.authProvider = authProvider
		log.Printf("Authentication enabled with type: %s", cfg.Security.Auth.Type)
	}

	// Configure encryption if enabled
	if cfg.Security.Encryption.Enabled {
		encProvider, err := security.NewEncryptionProvider(cfg.Security.Encryption)
		if err != nil {
			return nil, fmt.Errorf("error creating encryption provider: %v", err)
		}
		sender.encryptionProvider = encProvider
		log.Printf("Encryption enabled with type: %s", cfg.Security.Encryption.Type)
	}

	return sender, nil
}

// SetTelemetryTracer sets the OpenTelemetry tracer for the sender
func (s *HTTPSender) SetTelemetryTracer(tracer trace.Tracer) {
	s.tracer = tracer
}

// Start begins the sender process
func (s *HTTPSender) Start() {
	go s.flushLoop()
}

// Stop stops the sender and flushes any remaining logs
func (s *HTTPSender) Stop() {
	// Use a mutex to prevent double close
	s.lock.Lock()
	select {
	case <-s.stopCh:
		// Channel already closed, do nothing
		s.lock.Unlock()
		return
	default:
		close(s.stopCh)
		s.lock.Unlock()
	}
	<-s.stoppedCh
}

// Send adds a log line to the batch and triggers a flush if the batch is full
func (s *HTTPSender) Send(line string) {
	s.SendWithContext(context.Background(), line)
}

// SendWithContext adds a log line to the batch with tracing context and triggers a flush if the batch is full
func (s *HTTPSender) SendWithContext(ctx context.Context, line string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.batch = append(s.batch, line)
	if len(s.batch) >= s.batchSize {
		s.flushLockedWithContext(ctx)
	}
}

// flushLoop periodically flushes the batch based on the flush interval
func (s *HTTPSender) flushLoop() {
	// Ensure flush interval is positive
	interval := s.flushInterval
	if interval <= 0 {
		interval = 1 * time.Second // Default to 1 second if interval is invalid
	}

	ticker := time.NewTicker(interval)
	defer func() {
		ticker.Stop()
		s.flush() // Flush any remaining logs
		close(s.stoppedCh)
	}()

	for {
		select {
		case <-ticker.C:
			s.flush()
		case <-s.stopCh:
			return
		}
	}
}

// flush sends any pending log lines in the batch
func (s *HTTPSender) flush() {
	ctx := context.Background()
	if s.tracer != nil {
		var span trace.Span
		ctx, span = s.tracer.Start(ctx, "http_sender.flush")
		defer span.End()
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.flushLockedWithContext(ctx)
}

// flushLockedWithContext sends any pending log lines in the batch (must be called with lock held)
func (s *HTTPSender) flushLockedWithContext(ctx context.Context) {
	if len(s.batch) == 0 {
		return
	}

	// Create a copy of the batch to send
	toSend := make([]string, len(s.batch))
	copy(toSend, s.batch)
	s.batch = s.batch[:0] // Clear the batch but keep capacity

	// Send the batch asynchronously to avoid blocking
	go func(ctx context.Context, logs []string) {
		if err := s.sendBatchWithContext(ctx, logs); err != nil {
			log.Printf("Error sending batch: %v", err)
			// In a production system, we would queue for retry
		}
	}(ctx, toSend)
}

// sendBatchWithContext sends a batch of logs to the server with tracing context
func (s *HTTPSender) sendBatchWithContext(ctx context.Context, logs []string) error {
	// Create span for sending batch if tracer is available
	if s.tracer != nil {
		var span trace.Span
		ctx, span = s.tracer.Start(ctx, "http_sender.send_batch")
		defer span.End()

		// Add telemetry attributes
		span.SetAttributes(
			attribute.Int("batch.size", len(logs)),
			attribute.String("server.url", s.serverURL),
		)
	}

	// Marshal the logs to JSON
	data, err := json.Marshal(logs)
	if err != nil {
		if s.tracer != nil {
			trace.SpanFromContext(ctx).RecordError(err, trace.WithAttributes(
				attribute.String("error.type", "json_marshal"),
			))
		}
		return fmt.Errorf("error marshaling logs: %v", err)
	}

	// Encrypt data if encryption is enabled
	if s.encryptionProvider != nil {
		encryptedData, err := s.encryptionProvider.Encrypt(data)
		if err != nil {
			if s.tracer != nil {
				trace.SpanFromContext(ctx).RecordError(err, trace.WithAttributes(
					attribute.String("error.type", "encryption"),
				))
			}
			return fmt.Errorf("error encrypting data: %v", err)
		}
		data = encryptedData
	}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "POST", s.serverURL, bytes.NewBuffer(data))
	if err != nil {
		if s.tracer != nil {
			trace.SpanFromContext(ctx).RecordError(err, trace.WithAttributes(
				attribute.String("error.type", "create_request"),
			))
		}
		return fmt.Errorf("error creating request: %v", err)
	}

	// Set content type based on whether encryption is used
	if s.encryptionProvider != nil {
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("X-Encrypted", "true")
		req.Header.Set("X-Key-ID", s.encryptionProvider.GetKeyID())
	} else {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add authentication if configured
	if s.authProvider != nil {
		if err := s.authProvider.AddAuthentication(req); err != nil {
			if s.tracer != nil {
				trace.SpanFromContext(ctx).RecordError(err, trace.WithAttributes(
					attribute.String("error.type", "authentication"),
				))
			}
			return fmt.Errorf("error adding authentication: %v", err)
		}
	}

	// Send the request
	resp, err := s.client.Do(req)
	if err != nil {
		if s.tracer != nil {
			trace.SpanFromContext(ctx).RecordError(err, trace.WithAttributes(
				attribute.String("error.type", "http_request"),
			))
		}
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("server returned non-success status: %d", resp.StatusCode)
		if s.tracer != nil {
			trace.SpanFromContext(ctx).RecordError(err, trace.WithAttributes(
				attribute.String("error.type", "http_status"),
				attribute.Int("http.status_code", resp.StatusCode),
			))
		}
		return err
	}

	return nil
}

// Kept for backward compatibility
// sendBatch sends a batch of logs to the server
//
//nolint:unused,deadcode,golint,revive
func (s *HTTPSender) sendBatch(logs []string) error {
	return s.sendBatchWithContext(context.Background(), logs)
}
