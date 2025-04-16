package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/amirhossein-jamali/tailpost/pkg/config"
	httpserver "github.com/amirhossein-jamali/tailpost/pkg/http"
	"github.com/amirhossein-jamali/tailpost/pkg/observability"
	"github.com/amirhossein-jamali/tailpost/pkg/reader"
	"github.com/amirhossein-jamali/tailpost/pkg/sender"
	"github.com/amirhossein-jamali/tailpost/pkg/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Prometheus metrics
var (
	// Counter for total logs processed
	logsProcessedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tailpost_logs_processed_total",
			Help: "Total number of log lines processed",
		},
		[]string{"source_type"},
	)

	// Counter for logs sent successfully
	logsSentTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tailpost_logs_sent_total",
			Help: "Total number of log lines sent successfully",
		},
		[]string{"source_type"},
	)

	// Counter for log send failures
	logsSendFailuresTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tailpost_logs_send_failures_total",
			Help: "Total number of log send failures",
		},
		[]string{"source_type", "error_type"},
	)

	// Gauge for batch size
	batchSizeGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "tailpost_batch_size",
			Help: "Current batch size for log sending",
		},
	)

	// Histogram for send latency
	sendLatencyHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "tailpost_send_latency_seconds",
			Help:    "Latency of log sending operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"source_type"},
	)
)

func init() {
	// Register metrics with Prometheus
	prometheus.MustRegister(
		logsProcessedTotal,
		logsSentTotal,
		logsSendFailuresTotal,
		batchSizeGauge,
		sendLatencyHistogram,
	)
}

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to the configuration file")
	metricsAddr := flag.String("metrics-addr", ":8080", "The address to bind the metrics server to")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	logFormat := flag.String("log-format", "json", "Log format (json or console)")
	flag.Parse()

	// Configure structured logging
	var zapLevel zapcore.Level
	switch *logLevel {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	var encoderConfig zapcore.EncoderConfig
	if *logFormat == "console" {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	} else {
		encoderConfig = zap.NewProductionEncoderConfig()
	}
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if *logFormat == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		zapLevel,
	)
	logger := zap.New(core)
	defer logger.Sync()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	logger.Info("Loading configuration", zap.String("config_path", *configPath))
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Fatal("Error loading configuration", zap.Error(err))
	}
	logger.Info("Configuration loaded",
		zap.String("log_source_type", string(cfg.LogSourceType)),
		zap.String("server_url", cfg.ServerURL),
		zap.Int("batch_size", cfg.BatchSize),
		zap.Duration("flush_interval", cfg.FlushInterval))

	// Set the batch size gauge
	batchSizeGauge.Set(float64(cfg.BatchSize))

	// Log security configuration if enabled
	if cfg.Security.TLS.Enabled {
		logger.Info("TLS security is enabled")
	}
	if cfg.Security.Auth.Type != "none" {
		logger.Info("Authentication is enabled", zap.String("auth_type", cfg.Security.Auth.Type))
	}
	if cfg.Security.Encryption.Enabled {
		logger.Info("Encryption is enabled", zap.String("encryption_type", cfg.Security.Encryption.Type))
	}

	// Initialize telemetry if enabled
	var telemetryCleanup func()
	var telemetryManager *observability.TelemetryManager
	if cfg.Telemetry.Enabled {
		logger.Info("Initializing telemetry")

		// Configure OpenTelemetry
		telConfig := telemetry.Config{
			ServiceName:       cfg.Telemetry.ServiceName,
			ServiceVersion:    cfg.Telemetry.ServiceVersion,
			ExporterType:      cfg.Telemetry.ExporterType,
			ExporterEndpoint:  cfg.Telemetry.ExporterEndpoint,
			ExporterTimeout:   30 * time.Second,
			SamplingRate:      cfg.Telemetry.SamplingRate,
			PropagateContexts: true,
			Attributes:        cfg.Telemetry.Attributes,
			DisableTelemetry:  !cfg.Telemetry.Enabled,
		}

		var telErr error
		telemetryCleanup, telErr = telemetry.Setup(ctx, telConfig)
		if telErr != nil {
			logger.Warn("Failed to initialize OpenTelemetry", zap.Error(telErr))
		} else {
			logger.Info("OpenTelemetry initialized successfully")
		}

		// Initialize observability manager
		obsConfig := observability.TelemetryConfig{
			Enabled:            cfg.Telemetry.Enabled,
			ServiceName:        cfg.Telemetry.ServiceName,
			ServiceVersion:     cfg.Telemetry.ServiceVersion,
			ExporterType:       cfg.Telemetry.ExporterType,
			Endpoint:           cfg.Telemetry.ExporterEndpoint,
			SamplingRate:       cfg.Telemetry.SamplingRate,
			Headers:            cfg.Telemetry.Attributes,
			BatchTimeout:       5 * time.Second,
			MaxExportBatchSize: 512,
			MaxQueueSize:       2048,
		}

		telemetryManager = observability.NewTelemetryManager(obsConfig)
		if err := telemetryManager.Start(ctx); err != nil {
			logger.Warn("Failed to initialize observability manager", zap.Error(err))
		} else {
			logger.Info("Observability manager initialized successfully")
		}
	}

	// Create and start advanced metrics server with Prometheus handler
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	// Start health and metrics server with security if enabled
	var healthServer *httpserver.HealthServer
	if cfg.Security.TLS.Enabled || cfg.Security.Auth.Type != "none" {
		// Create secure health server
		secureHealthServer, err := httpserver.NewSecureHealthServer(*metricsAddr, cfg.Security)
		if err != nil {
			logger.Fatal("Error creating secure health server", zap.Error(err))
		}
		healthServer = secureHealthServer
	} else {
		// Create standard health server
		healthServer = httpserver.NewHealthServer(*metricsAddr)
	}

	// Start the health server
	if err := healthServer.Start(); err != nil {
		logger.Fatal("Error starting health server", zap.Error(err))
	}

	// Register Prometheus handler with the health server
	http.Handle("/metrics", promhttp.Handler())

	// Create components
	var logReader reader.LogReader

	// Create span for reader initialization if telemetry is available
	var initSpan trace.Span
	if telemetryManager != nil {
		tracer := telemetry.Tracer("tailpost.agent")
		_, initSpan = tracer.Start(ctx, "init_reader")
		defer initSpan.End()
	}

	// Determine if we're using a file reader or other type of reader
	if cfg.LogSourceType != "" {
		sourceType, err := reader.ParseSourceType(string(cfg.LogSourceType))
		if err != nil {
			logger.Fatal("Error parsing log source type", zap.Error(err))
		}

		logger.Debug("Creating reader for source type", zap.String("source_type", string(sourceType)))

		// Convert map selectors to string selectors if needed
		podSelector := ""
		if len(cfg.PodSelector) > 0 {
			// Convert map to string in format "key1=value1,key2=value2"
			selectors := make([]string, 0, len(cfg.PodSelector))
			for k, v := range cfg.PodSelector {
				selectors = append(selectors, k+"="+v)
			}
			podSelector = strings.Join(selectors, ",")
		}

		namespaceSelector := ""
		if len(cfg.NamespaceSelector) > 0 {
			// Convert map to string in format "key1=value1,key2=value2"
			selectors := make([]string, 0, len(cfg.NamespaceSelector))
			for k, v := range cfg.NamespaceSelector {
				selectors = append(selectors, k+"="+v)
			}
			namespaceSelector = strings.Join(selectors, ",")
		}

		sourceConfig := reader.LogSourceConfig{
			Type:                 sourceType,
			Path:                 cfg.LogPath,
			Namespace:            cfg.Namespace,
			PodName:              cfg.PodName,
			ContainerName:        cfg.ContainerName,
			PodSelector:          podSelector,
			NamespaceSelector:    namespaceSelector,
			WindowsEventLogName:  cfg.WindowsEventLogName,
			WindowsEventLogLevel: cfg.WindowsEventLogLevel,
			MacOSLogQuery:        cfg.MacOSLogQuery,
		}

		// Add platform-specific logging
		switch sourceType {
		case reader.WindowsEventSourceType:
			logger.Info("Initializing Windows Event Log reader",
				zap.String("log_name", cfg.WindowsEventLogName),
				zap.String("min_level", cfg.WindowsEventLogLevel))
		case reader.MacOSASLSourceType:
			logger.Info("Initializing macOS ASL log reader",
				zap.String("query", cfg.MacOSLogQuery))
		case reader.FileSourceType:
			logger.Info("Initializing file log reader",
				zap.String("path", cfg.LogPath))
		case reader.ContainerSourceType:
			logger.Info("Initializing Kubernetes container log reader",
				zap.String("namespace", cfg.Namespace),
				zap.String("pod", cfg.PodName),
				zap.String("container", cfg.ContainerName))
		}

		logReader, err = reader.NewReader(sourceConfig)
		if err != nil {
			logger.Fatal("Error creating reader", zap.Error(err))
		}
	} else {
		// Default to file reader for backward compatibility
		logger.Info("Using default file reader", zap.String("path", cfg.LogPath))
		logReader = reader.NewFileReader(cfg.LogPath)
	}

	// Create secure sender with TLS and authentication if enabled
	var httpSender *sender.HTTPSender
	if cfg.Security.TLS.Enabled || cfg.Security.Auth.Type != "none" || cfg.Security.Encryption.Enabled {
		secureHTTPSender, err := sender.NewSecureHTTPSender(cfg)
		if err != nil {
			logger.Fatal("Error creating secure HTTP sender", zap.Error(err))
		}
		httpSender = secureHTTPSender
	} else {
		// Create standard HTTP sender
		httpSender = sender.NewHTTPSender(cfg.ServerURL, cfg.BatchSize, cfg.FlushInterval)
	}

	// Set telemetry tracer if available
	if telemetryManager != nil {
		httpSender.SetTelemetryTracer(telemetryManager.Tracer())
	}

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start components
	logger.Info("Starting reader")
	if err := logReader.Start(); err != nil {
		logger.Fatal("Error starting reader", zap.Error(err))
	}

	logger.Info("Starting HTTP sender")
	httpSender.Start()

	// Use a WaitGroup to ensure clean shutdown
	var wg sync.WaitGroup
	wg.Add(1)

	// Connect reader to sender
	go func() {
		defer wg.Done()

		sourceType := string(cfg.LogSourceType)
		lineCount := 0

		for {
			select {
			case <-ctx.Done():
				logger.Info("Stopping log processing due to context cancellation")
				return
			case line, ok := <-logReader.Lines():
				if !ok {
					logger.Info("Log reader channel closed, stopping processing")
					return
				}

				// Increment the processed logs counter
				logsProcessedTotal.WithLabelValues(sourceType).Inc()

				// Track processing in telemetry if enabled
				startTime := time.Now()

				if telemetryManager != nil {
					lineCtx, processSpan := telemetryManager.Tracer().Start(ctx, "process_log_line")
					httpSender.SendWithContext(lineCtx, line)
					processSpan.End()
				} else {
					httpSender.Send(line)
				}

				// Record metrics for the send operation
				duration := time.Since(startTime).Seconds()
				sendLatencyHistogram.WithLabelValues(sourceType).Observe(duration)

				// We can't track actual send success/failure from here
				// but we could add a method to HTTPSender to expose this data
				logsSentTotal.WithLabelValues(sourceType).Inc()

				lineCount++
				if lineCount%1000 == 0 {
					logger.Info("Processed log lines", zap.Int("count", lineCount))
				}
			}
		}
	}()

	logger.Info("Tailpost agent started successfully")
	// Mark as ready
	healthServer.SetReady(true)

	// Wait for shutdown signal
	sig := <-sigCh
	logger.Info("Received signal, shutting down", zap.String("signal", sig.String()))

	// Cancel the context to notify all goroutines
	cancel()

	// Mark as not ready
	healthServer.SetReady(false)

	// Set a timeout for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop components in reverse order
	logger.Info("Stopping health server")
	healthServer.Stop()

	logger.Info("Stopping sender")
	httpSender.Stop()

	logger.Info("Stopping reader")
	logReader.Stop()

	// Wait for processing to complete
	logger.Info("Waiting for all operations to complete")
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("All operations completed successfully")
	case <-shutdownCtx.Done():
		logger.Warn("Shutdown timed out, some operations may not have completed")
	}

	// Shutdown telemetry managers if initialized
	if telemetryManager != nil {
		logger.Info("Shutting down observability manager")
		if err := telemetryManager.Shutdown(shutdownCtx); err != nil {
			logger.Warn("Error shutting down observability manager", zap.Error(err))
		}
	}

	// Clean up OpenTelemetry if initialized
	if telemetryCleanup != nil {
		logger.Info("Shutting down OpenTelemetry")
		telemetryCleanup()
	}

	logger.Info("Shutdown complete")
}
