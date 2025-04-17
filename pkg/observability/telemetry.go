package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// TelemetryConfig configures the OpenTelemetry integration
type TelemetryConfig struct {
	// Enabled determines if OpenTelemetry tracing is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`

	// ServiceName is used to identify this service in traces
	ServiceName string `json:"service_name" yaml:"service_name"`

	// ServiceVersion is the version of the service
	ServiceVersion string `json:"service_version" yaml:"service_version"`

	// ExporterType is the type of exporter to use (grpc, http, or none for no exporter)
	ExporterType string `json:"exporter_type" yaml:"exporter_type"`

	// Endpoint is the OpenTelemetry collector endpoint
	Endpoint string `json:"endpoint" yaml:"endpoint"`

	// Headers to include in OTLP exports
	Headers map[string]string `json:"headers" yaml:"headers"`

	// SamplingRate controls how many traces are sampled (0.0-1.0)
	SamplingRate float64 `json:"sampling_rate" yaml:"sampling_rate"`

	// BatchTimeout is how often the processor should send spans
	BatchTimeout time.Duration `json:"batch_timeout" yaml:"batch_timeout"`

	// MaxExportBatchSize is the maximum number of spans to export at once
	MaxExportBatchSize int `json:"max_export_batch_size" yaml:"max_export_batch_size"`

	// MaxQueueSize is the maximum queue size for pending spans
	MaxQueueSize int `json:"max_queue_size" yaml:"max_queue_size"`
}

// SetDefaults sets default values for TelemetryConfig if not set
func (c *TelemetryConfig) SetDefaults() {
	if c.ServiceName == "" {
		c.ServiceName = "tailpost-agent"
	}
	if c.ServiceVersion == "" {
		c.ServiceVersion = "unknown"
	}
	if c.ExporterType == "" {
		c.ExporterType = "grpc"
	}
	if c.Endpoint == "" {
		c.Endpoint = "localhost:4317"
	}
	if c.SamplingRate == 0 {
		c.SamplingRate = 0.1 // Default to sampling 10% of traces
	}
	if c.BatchTimeout == 0 {
		c.BatchTimeout = 5 * time.Second
	}
	if c.MaxExportBatchSize == 0 {
		c.MaxExportBatchSize = 512
	}
	if c.MaxQueueSize == 0 {
		c.MaxQueueSize = 2048
	}
	if c.Headers == nil {
		c.Headers = make(map[string]string)
	}
}

// TelemetryManager manages OpenTelemetry integration
type TelemetryManager struct {
	config TelemetryConfig
	tp     *sdktrace.TracerProvider
	tracer trace.Tracer
}

// NewTelemetryManager creates a new TelemetryManager with the given configuration
func NewTelemetryManager(config TelemetryConfig) *TelemetryManager {
	config.SetDefaults()
	return &TelemetryManager{
		config: config,
	}
}

// Start initializes the OpenTelemetry integration
func (tm *TelemetryManager) Start(ctx context.Context) error {
	if !tm.config.Enabled {
		// Use a no-op tracer if telemetry is disabled
		tm.tracer = noop.NewTracerProvider().Tracer("tailpost")
		return nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(tm.config.ServiceName),
			semconv.ServiceVersion(tm.config.ServiceVersion),
			attribute.String("environment", "production"),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Create exporter based on configuration
	var exporter sdktrace.SpanExporter
	var exportErr error

	switch tm.config.ExporterType {
	case "grpc":
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(tm.config.Endpoint),
			otlptracegrpc.WithTimeout(30 * time.Second),
		}

		if tm.config.Headers != nil {
			headers := make(map[string]string)
			for k, v := range tm.config.Headers {
				headers[k] = v
			}
			opts = append(opts, otlptracegrpc.WithHeaders(headers))
		}

		exporter, exportErr = otlptrace.New(ctx, otlptracegrpc.NewClient(opts...))
	case "http":
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(tm.config.Endpoint),
			otlptracehttp.WithTimeout(30 * time.Second),
		}

		if tm.config.Headers != nil {
			headers := make(map[string]string)
			for k, v := range tm.config.Headers {
				headers[k] = v
			}
			opts = append(opts, otlptracehttp.WithHeaders(headers))
		}

		exporter, exportErr = otlptrace.New(ctx, otlptracehttp.NewClient(opts...))
	case "none":
		// No exporter, use no-op
		tm.tracer = noop.NewTracerProvider().Tracer("tailpost")
		return nil
	default:
		return fmt.Errorf("unknown exporter type: %s", tm.config.ExporterType)
	}

	if exportErr != nil {
		return fmt.Errorf("failed to create exporter: %w", exportErr)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter,
			sdktrace.WithMaxExportBatchSize(tm.config.MaxExportBatchSize),
			sdktrace.WithBatchTimeout(tm.config.BatchTimeout),
			sdktrace.WithMaxQueueSize(tm.config.MaxQueueSize),
		),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(tm.config.SamplingRate)),
	)

	// Set as global trace provider
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tm.tp = tp
	tm.tracer = tp.Tracer("tailpost")
	return nil
}

// Shutdown shuts down the telemetry manager
func (tm *TelemetryManager) Shutdown(ctx context.Context) error {
	if tm.tp != nil {
		return tm.tp.Shutdown(ctx)
	}
	return nil
}

// Tracer returns the OpenTelemetry tracer
func (tm *TelemetryManager) Tracer() trace.Tracer {
	return tm.tracer
}

// StartSpan starts a new span with the given name and returns the span and context
func (tm *TelemetryManager) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return tm.tracer.Start(ctx, name, opts...)
}

// AddEventToSpan adds an event to the current span
func (tm *TelemetryManager) AddEventToSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetSpanAttributes sets attributes on the current span
func (tm *TelemetryManager) SetSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError records an error on the current span
func (tm *TelemetryManager) RecordError(ctx context.Context, err error, opts ...trace.EventOption) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err, opts...)
}
