package telemetry

import (
	"context"
	"log"
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
)

// Config holds the configuration for the telemetry setup
type Config struct {
	ServiceName       string
	ServiceVersion    string
	ExporterType      string // "grpc" or "http"
	ExporterEndpoint  string
	ExporterTimeout   time.Duration
	SamplingRate      float64
	PropagateContexts bool
	Attributes        map[string]string
	DisableTelemetry  bool
}

// DefaultConfig returns a default configuration for telemetry
func DefaultConfig() Config {
	return Config{
		ServiceName:       "tailpost",
		ServiceVersion:    "0.1.0",
		ExporterType:      "http",
		ExporterEndpoint:  "http://localhost:4318",
		ExporterTimeout:   30 * time.Second,
		SamplingRate:      1.0, // Always sample
		PropagateContexts: true,
		Attributes:        map[string]string{},
		DisableTelemetry:  false,
	}
}

// Setup initializes the OpenTelemetry SDK with the provided configuration
func Setup(ctx context.Context, cfg Config) (func(), error) {
	if cfg.DisableTelemetry {
		return func() {}, nil
	}

	// Create resource with service information and additional attributes
	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(cfg.ServiceName),
		semconv.ServiceVersionKey.String(cfg.ServiceVersion),
	}

	for k, v := range cfg.Attributes {
		attrs = append(attrs, attribute.String(k, v))
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(attrs...),
		resource.WithProcessRuntimeDescription(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, err
	}

	// Configure trace exporter
	var exporter *otlptrace.Exporter
	if cfg.ExporterType == "grpc" {
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.ExporterEndpoint),
			otlptracegrpc.WithTimeout(cfg.ExporterTimeout),
		}
		client := otlptracegrpc.NewClient(opts...)
		exporter, err = otlptrace.New(ctx, client)
	} else {
		// Default to HTTP
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(cfg.ExporterEndpoint),
			otlptracehttp.WithTimeout(cfg.ExporterTimeout),
		}
		client := otlptracehttp.NewClient(opts...)
		exporter, err = otlptrace.New(ctx, client)
	}

	if err != nil {
		return nil, err
	}

	// Configure trace provider
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SamplingRate)),
	)

	// Set global trace provider
	otel.SetTracerProvider(traceProvider)

	// Set global propagator if enabled
	if cfg.PropagateContexts {
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
	}

	// Return a cleanup function
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := traceProvider.Shutdown(ctx); err != nil {
			// Log error using standard library since we're in cleanup
			log.Printf("Error shutting down trace provider: %v", err)
		}
	}, nil
}

// Tracer returns a named tracer from the global provider
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
