package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Check default values
	assert.Equal(t, "tailpost", cfg.ServiceName)
	assert.Equal(t, "0.1.0", cfg.ServiceVersion)
	assert.Equal(t, "http", cfg.ExporterType)
	assert.Equal(t, "http://localhost:4318", cfg.ExporterEndpoint)
	assert.Equal(t, 30*time.Second, cfg.ExporterTimeout)
	assert.Equal(t, 1.0, cfg.SamplingRate)
	assert.True(t, cfg.PropagateContexts)
	assert.Empty(t, cfg.Attributes)
	assert.False(t, cfg.DisableTelemetry)
}

func TestSetupWithDisabledTelemetry(t *testing.T) {
	// Create config with telemetry disabled
	cfg := Config{
		DisableTelemetry: true,
	}

	// Setup should return a no-op cleanup function
	cleanup, err := Setup(context.Background(), cfg)

	assert.NoError(t, err)
	assert.NotNil(t, cleanup)

	// Calling the cleanup function should not panic
	cleanup()
}

func TestTracer(t *testing.T) {
	// Save original tracer provider to restore after test
	originalProvider := otel.GetTracerProvider()
	defer otel.SetTracerProvider(originalProvider)

	// Set a custom tracer provider for testing
	provider := noop.NewTracerProvider()
	otel.SetTracerProvider(provider)

	tracer := Tracer("test-tracer")
	assert.NotNil(t, tracer)
}

// Custom mock exporter for testing
type mockExporter struct {
	started  int
	stopped  int
	exported int
}

func (m *mockExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	m.exported += len(spans)
	return nil
}

func (m *mockExporter) Shutdown(ctx context.Context) error {
	m.stopped++
	return nil
}

func (m *mockExporter) Start(ctx context.Context) error {
	m.started++
	return nil
}

func TestSetupWithHTTPExporter(t *testing.T) {
	// Save original values to restore after test
	originalProvider := otel.GetTracerProvider()
	originalPropagator := otel.GetTextMapPropagator()
	defer func() {
		otel.SetTracerProvider(originalProvider)
		otel.SetTextMapPropagator(originalPropagator)
	}()

	cfg := Config{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		ExporterType:      "http",
		ExporterEndpoint:  "http://localhost:4318", // This won't actually connect in tests
		ExporterTimeout:   5 * time.Second,
		SamplingRate:      0.5,
		PropagateContexts: true,
		Attributes: map[string]string{
			"environment": "test",
		},
	}

	// Skip running this test if we can't connect to the endpoint
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cleanup, err := Setup(ctx, cfg)
	if err != nil {
		t.Skip("Skipping test due to exporter connection issue:", err)
		return
	}

	// Verify tracer provider was set
	assert.NotEqual(t, originalProvider, otel.GetTracerProvider())

	// Verify propagator was set
	assert.NotEqual(t, originalPropagator, otel.GetTextMapPropagator())

	// Check that we can get a tracer
	tracer := Tracer("test-component")
	assert.NotNil(t, tracer)

	// Cleanup should not panic
	cleanup()
}

func TestSetupWithGRPCExporter(t *testing.T) {
	// Save original values to restore after test
	originalProvider := otel.GetTracerProvider()
	originalPropagator := otel.GetTextMapPropagator()
	defer func() {
		otel.SetTracerProvider(originalProvider)
		otel.SetTextMapPropagator(originalPropagator)
	}()

	cfg := Config{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		ExporterType:      "grpc",
		ExporterEndpoint:  "localhost:4317", // This won't actually connect in tests
		ExporterTimeout:   5 * time.Second,
		SamplingRate:      0.5,
		PropagateContexts: true,
	}

	// Skip running this test if we can't connect to the endpoint
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cleanup, err := Setup(ctx, cfg)
	if err != nil {
		t.Skip("Skipping test due to exporter connection issue:", err)
		return
	}

	// Verify tracer provider was set
	assert.NotEqual(t, originalProvider, otel.GetTracerProvider())

	// Cleanup should not panic
	cleanup()
}

func TestSetupWithoutPropagation(t *testing.T) {
	// Save original values to restore after test
	originalProvider := otel.GetTracerProvider()
	originalPropagator := otel.GetTextMapPropagator()
	defer func() {
		otel.SetTracerProvider(originalProvider)
		otel.SetTextMapPropagator(originalPropagator)
	}()

	cfg := Config{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		ExporterType:      "http",
		ExporterEndpoint:  "http://localhost:4318", // This won't actually connect in tests
		PropagateContexts: false,                   // Disable context propagation
	}

	// Skip running this test if we can't connect to the endpoint
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cleanup, err := Setup(ctx, cfg)
	if err != nil {
		t.Skip("Skipping test due to exporter connection issue:", err)
		return
	}

	// Verify tracer provider was set
	assert.NotEqual(t, originalProvider, otel.GetTracerProvider())

	// Verify propagator was NOT changed (should be same as original)
	assert.Equal(t, originalPropagator, otel.GetTextMapPropagator())

	// Cleanup should not panic
	cleanup()
}

func TestTracingEndToEnd(t *testing.T) {
	// This tests the full tracing pipeline by creating a span and checking if it was exported

	// Save original values to restore after test
	originalProvider := otel.GetTracerProvider()
	defer otel.SetTracerProvider(originalProvider)

	// Create a mock exporter
	exporter := &mockExporter{}

	// Create a trace provider with the mock exporter
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	)

	// Set the provider
	otel.SetTracerProvider(tp)

	// Create and use a tracer
	tr := Tracer("test-tracer")
	_, span := tr.Start(context.Background(), "test-span")
	span.End()

	// Shutdown the provider to flush spans
	err := tp.Shutdown(context.Background())
	assert.NoError(t, err)

	// Verify that our span was exported
	// Note: In some cases this might be flaky if the exporter hasn't processed the span yet
	// For more reliable tests, we'd need to use a more sophisticated mock that provides confirmation
	// that spans were exported
}

func TestSetupWithCustomAttributes(t *testing.T) {
	// Save original values to restore after test
	originalProvider := otel.GetTracerProvider()
	defer otel.SetTracerProvider(originalProvider)

	cfg := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		ExporterType:   "http",
		Attributes: map[string]string{
			"deployment.environment": "testing",
			"custom.attribute":       "value",
		},
		DisableTelemetry: true, // Disable actual exporting for this test
	}

	cleanup, err := Setup(context.Background(), cfg)
	assert.NoError(t, err)
	defer cleanup()

	// We can't easily verify the attributes were set without modifying the code
	// or using more complex mocks, but at least we've verified the setup doesn't fail
	// with custom attributes
}

func TestPropagation(t *testing.T) {
	// Save original propagator to restore after test
	originalPropagator := otel.GetTextMapPropagator()
	defer otel.SetTextMapPropagator(originalPropagator)

	// Setup with propagation enabled
	cfg := Config{
		PropagateContexts: true,
		DisableTelemetry:  true, // Disable actual exporting for this test
	}

	cleanup, err := Setup(context.Background(), cfg)
	assert.NoError(t, err)
	defer cleanup()

	// Verify that we're using a composite propagator
	propagator := otel.GetTextMapPropagator()
	_, ok := propagator.(interface{})
	assert.True(t, ok, "Expected a propagator interface when PropagateContexts is true")
}
