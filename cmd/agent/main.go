package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amirhossein-jamali/tailpost/pkg/config"
	httpserver "github.com/amirhossein-jamali/tailpost/pkg/http"
	"github.com/amirhossein-jamali/tailpost/pkg/observability"
	"github.com/amirhossein-jamali/tailpost/pkg/reader"
	"github.com/amirhossein-jamali/tailpost/pkg/sender"
	"github.com/amirhossein-jamali/tailpost/pkg/telemetry"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Main entry point for the TailPost agent application
func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to the configuration file")
	metricsAddr := flag.String("metrics-addr", ":8080", "The address to bind the metrics server to")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	logFormat := flag.String("log-format", "json", "Log format (json or console)")
	flag.Parse()

	// Configure logging
	logger := setupLogging(*logLevel, *logFormat)
	defer func() {
		if err := logger.Sync(); err != nil {
			// Can't use logger itself to log this error
			fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
		}
	}()

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

	// Initialize telemetry if enabled
	var telemetryCleanup func()
	var telemetryManager *observability.TelemetryManager
	if cfg.Telemetry.Enabled {
		telemetryManager, telemetryCleanup = setupTelemetry(ctx, cfg, logger)
		defer telemetryCleanup()
	}

	// Start health and metrics server
	healthServer := setupHealthServer(*metricsAddr, cfg, logger)

	// Create log reader
	logReader, err := setupLogReader(ctx, cfg, logger, telemetryManager)
	if err != nil {
		logger.Fatal("Error creating log reader", zap.Error(err))
	}

	// Create log sender
	logSender, err := setupLogSender(cfg, logger, telemetryManager)
	if err != nil {
		logger.Fatal("Error creating log sender", zap.Error(err))
	}

	// Start processing logs
	processingDone := make(chan struct{})
	go processLogs(ctx, logReader, logSender, logger, processingDone)

	// Handle signals for graceful shutdown
	handleSignals(ctx, cancel, logReader, logSender, healthServer, logger, processingDone)
}

// setupLogging configures the logger based on the provided level and format
func setupLogging(logLevel, logFormat string) *zap.Logger {
	// Configure based on the log level and format
	var zapLevel zapcore.Level
	switch logLevel {
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
	if logFormat == "console" {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	} else {
		encoderConfig = zap.NewProductionEncoderConfig()
	}
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if logFormat == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		zapLevel,
	)
	return zap.New(core)
}

// setupTelemetry initializes the telemetry system if enabled
func setupTelemetry(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*observability.TelemetryManager, func()) {
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

	cleanup, err := telemetry.Setup(ctx, telConfig)
	if err != nil {
		logger.Warn("Failed to initialize OpenTelemetry", zap.Error(err))
		return nil, func() {}
	}
	logger.Info("OpenTelemetry initialized successfully")

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

	telemetryManager := observability.NewTelemetryManager(obsConfig)
	if err := telemetryManager.Start(ctx); err != nil {
		logger.Warn("Failed to initialize observability manager", zap.Error(err))
	} else {
		logger.Info("Observability manager initialized successfully")
	}

	return telemetryManager, cleanup
}

// setupHealthServer sets up and starts the health and metrics server
func setupHealthServer(metricsAddr string, cfg *config.Config, logger *zap.Logger) *httpserver.HealthServer {
	// Create and start health server with security if enabled
	var healthServer *httpserver.HealthServer
	if cfg.Security.TLS.Enabled || cfg.Security.Auth.Type != "none" {
		// Create secure health server
		secureHealthServer, err := httpserver.NewSecureHealthServer(metricsAddr, cfg.Security)
		if err != nil {
			logger.Fatal("Error creating secure health server", zap.Error(err))
		}
		healthServer = secureHealthServer
	} else {
		// Create standard health server
		healthServer = httpserver.NewHealthServer(metricsAddr)
	}

	// Start the health server
	if err := healthServer.Start(); err != nil {
		logger.Fatal("Error starting health server", zap.Error(err))
	}

	// Register Prometheus handler with the health server
	http.Handle("/metrics", promhttp.Handler())

	return healthServer
}

// setupLogReader creates and configures the appropriate log reader
func setupLogReader(ctx context.Context, cfg *config.Config, logger *zap.Logger, telemetryManager *observability.TelemetryManager) (reader.LogReader, error) {
	// Create log reader configuration
	sourceConfig := reader.LogSourceConfig{
		Type:                 reader.LogSourceType(cfg.LogSourceType),
		Path:                 cfg.LogPath,
		Namespace:            cfg.Namespace,
		PodName:              cfg.PodName,
		ContainerName:        cfg.ContainerName,
		WindowsEventLogName:  cfg.WindowsEventLogName,
		WindowsEventLogLevel: cfg.WindowsEventLogLevel,
		MacOSLogQuery:        cfg.MacOSLogQuery,
	}

	// Create the log reader
	logReader, err := reader.NewReader(sourceConfig)
	if err != nil {
		return nil, err
	}

	// Start the log reader
	if err := logReader.Start(); err != nil {
		return nil, err
	}

	return logReader, nil
}

// setupLogSender creates and configures the log sender
func setupLogSender(cfg *config.Config, logger *zap.Logger, telemetryManager *observability.TelemetryManager) (*sender.HTTPSender, error) {
	var logSender *sender.HTTPSender
	var err error

	// Create sender based on configuration
	if cfg.Security.TLS.Enabled || cfg.Security.Auth.Type != "none" || cfg.Security.Encryption.Enabled {
		logSender, err = sender.NewSecureHTTPSender(cfg)
		if err != nil {
			return nil, err
		}
	} else {
		logSender = sender.NewHTTPSender(cfg.ServerURL, cfg.BatchSize, cfg.FlushInterval)
	}

	// Configure telemetry for the sender if available
	if telemetryManager != nil {
		tracer := telemetry.Tracer("tailpost.sender")
		logSender.SetTelemetryTracer(tracer)
	}

	// Start the sender
	logSender.Start()

	return logSender, nil
}

// processLogs processes logs from the reader and sends them through the sender
func processLogs(ctx context.Context, logReader reader.LogReader, logSender *sender.HTTPSender, logger *zap.Logger, done chan struct{}) {
	logger.Info("Starting log processing")

	// Process logs until context is cancelled
	linesCh := logReader.Lines()
	for {
		select {
		case <-ctx.Done():
			logger.Info("Log processing stopped due to context cancellation")
			close(done)
			return
		case line, ok := <-linesCh:
			if !ok {
				logger.Info("Log reader channel closed")
				close(done)
				return
			}

			// Process and send the log
			logSender.SendWithContext(ctx, line)
		}
	}
}

// handleSignals sets up signal handling for graceful shutdown
func handleSignals(ctx context.Context, cancel context.CancelFunc, logReader reader.LogReader,
	logSender *sender.HTTPSender, healthServer *httpserver.HealthServer,
	logger *zap.Logger, processingDone chan struct{}) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal
	sig := <-sigCh
	logger.Info("Received signal, shutting down", zap.String("signal", sig.String()))

	// Cancel context to stop all operations
	cancel()

	// Stop components gracefully
	logger.Info("Stopping log reader")
	logReader.Stop()

	logger.Info("Stopping log sender")
	logSender.Stop()

	logger.Info("Stopping health server")
	if err := healthServer.Stop(); err != nil {
		logger.Error("Error stopping health server", zap.Error(err))
	}

	// Wait for processing to complete
	<-processingDone
	logger.Info("Shutdown complete")
}
