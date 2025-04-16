package observability

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TestTelemetryConfig_SetDefaults(t *testing.T) {
	// Test with empty config
	emptyConfig := TelemetryConfig{}
	emptyConfig.SetDefaults()

	// Verify default values
	assert.Equal(t, "tailpost-agent", emptyConfig.ServiceName)
	assert.Equal(t, "unknown", emptyConfig.ServiceVersion)
	assert.Equal(t, "grpc", emptyConfig.ExporterType)
	assert.Equal(t, "localhost:4317", emptyConfig.Endpoint)
	assert.Equal(t, 0.1, emptyConfig.SamplingRate)
	assert.Equal(t, 5*time.Second, emptyConfig.BatchTimeout)
	assert.Equal(t, 512, emptyConfig.MaxExportBatchSize)
	assert.Equal(t, 2048, emptyConfig.MaxQueueSize)
	assert.NotNil(t, emptyConfig.Headers)

	// Test with partially configured config
	partialConfig := TelemetryConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		SamplingRate:   0.5,
	}
	partialConfig.SetDefaults()

	// Verify custom values are preserved and defaults are set for unspecified fields
	assert.Equal(t, "test-service", partialConfig.ServiceName)
	assert.Equal(t, "1.0.0", partialConfig.ServiceVersion)
	assert.Equal(t, "grpc", partialConfig.ExporterType)
	assert.Equal(t, "localhost:4317", partialConfig.Endpoint)
	assert.Equal(t, 0.5, partialConfig.SamplingRate)
	assert.Equal(t, 5*time.Second, partialConfig.BatchTimeout)
	assert.Equal(t, 512, partialConfig.MaxExportBatchSize)
	assert.Equal(t, 2048, partialConfig.MaxQueueSize)
}

func TestNewTelemetryManager(t *testing.T) {
	config := TelemetryConfig{
		Enabled:     true,
		ServiceName: "test-service",
	}

	tm := NewTelemetryManager(config)
	assert.NotNil(t, tm)
	assert.Equal(t, config.ServiceName, tm.config.ServiceName)

	// Verify that SetDefaults was called
	assert.Equal(t, "grpc", tm.config.ExporterType)
}

func TestTelemetryManager_Start_Disabled(t *testing.T) {
	config := TelemetryConfig{
		Enabled: false,
	}

	tm := NewTelemetryManager(config)
	err := tm.Start(context.Background())

	// No error should occur and a no-op tracer should be created
	assert.NoError(t, err)
	assert.NotNil(t, tm.tracer)
}

func TestTelemetryManager_Start_InvalidExporter(t *testing.T) {
	config := TelemetryConfig{
		Enabled:      true,
		ExporterType: "invalid",
	}

	tm := NewTelemetryManager(config)
	err := tm.Start(context.Background())

	// Should return invalid exporter type error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown exporter type")
}

func TestTelemetryManager_Start_NoExporter(t *testing.T) {
	config := TelemetryConfig{
		Enabled:      true,
		ExporterType: "none",
	}

	tm := NewTelemetryManager(config)
	err := tm.Start(context.Background())

	// No error should occur and a no-op tracer should be created
	assert.NoError(t, err)
	assert.NotNil(t, tm.tracer)
}

func TestTelemetryManager_Shutdown(t *testing.T) {
	config := TelemetryConfig{
		Enabled: false,
	}

	tm := NewTelemetryManager(config)
	err := tm.Start(context.Background())
	assert.NoError(t, err)

	// For disabled telemetry, shutdown should not error
	err = tm.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestTelemetryManager_Tracer(t *testing.T) {
	config := TelemetryConfig{
		Enabled: false,
	}

	tm := NewTelemetryManager(config)
	err := tm.Start(context.Background())
	assert.NoError(t, err)

	// Should return a valid tracer
	tracer := tm.Tracer()
	assert.NotNil(t, tracer)
}

func TestTelemetryManager_StartSpan(t *testing.T) {
	config := TelemetryConfig{
		Enabled: false,
	}

	tm := NewTelemetryManager(config)
	err := tm.Start(context.Background())
	assert.NoError(t, err)

	// Create a new span
	ctx, span := tm.StartSpan(context.Background(), "test-span")
	assert.NotNil(t, ctx)
	assert.NotNil(t, span)

	// Close the span
	span.End()
}

func TestTelemetryManager_AddEventToSpan(t *testing.T) {
	config := TelemetryConfig{
		Enabled: false,
	}

	tm := NewTelemetryManager(config)
	err := tm.Start(context.Background())
	assert.NoError(t, err)

	// Create a new span
	ctx, span := tm.StartSpan(context.Background(), "test-span")

	// Add an event to the span
	attrs := []attribute.KeyValue{
		attribute.String("key", "value"),
	}
	tm.AddEventToSpan(ctx, "test-event", attrs...)

	// Close the span
	span.End()
}

func TestTelemetryManager_SetSpanAttributes(t *testing.T) {
	config := TelemetryConfig{
		Enabled: false,
	}

	tm := NewTelemetryManager(config)
	err := tm.Start(context.Background())
	assert.NoError(t, err)

	// Create a new span
	ctx, span := tm.StartSpan(context.Background(), "test-span")

	// Set span attributes
	attrs := []attribute.KeyValue{
		attribute.String("key", "value"),
		attribute.Int("count", 42),
	}
	tm.SetSpanAttributes(ctx, attrs...)

	// Close the span
	span.End()
}

func TestTelemetryManager_RecordError(t *testing.T) {
	config := TelemetryConfig{
		Enabled: false,
	}

	tm := NewTelemetryManager(config)
	err := tm.Start(context.Background())
	assert.NoError(t, err)

	// Create a new span
	ctx, span := tm.StartSpan(context.Background(), "test-span")

	// Record an error on the span
	testErr := assert.AnError
	tm.RecordError(ctx, testErr, trace.WithStackTrace(true))

	// Close the span
	span.End()
}

// This test requires mocking the OpenTelemetry exporter
// Here we're just testing the main path without an actual connection
func TestTelemetryManager_Start_WithGRPC(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := TelemetryConfig{
		Enabled:      true,
		ExporterType: "grpc",
		Endpoint:     "localhost:4317", // Assuming no OpenTelemetry collector is available
		Headers: map[string]string{
			"test-header": "test-value",
		},
	}

	tm := NewTelemetryManager(config)
	// Due to absence of actual collector, expecting error
	_ = tm.Start(context.Background())

	// Regardless of the outcome, just ensure the process started
	// If run in an environment with a real collector, this should succeed
	if tm.tp != nil {
		_ = tm.Shutdown(context.Background())
	}
}
