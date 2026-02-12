package telemetry

import (
	"context"
	"fmt"
	"os"
	"time"

	"distributed-crawler/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	otelTrace "go.opentelemetry.io/otel/trace"
	otelMetric "go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TelemetryProvider holds the OpenTelemetry providers and shutdown functions
type TelemetryProvider struct {
	tracerProvider *trace.TracerProvider
	meterProvider  *metric.MeterProvider
	shutdownFuncs  []func(context.Context) error
}

// NewTelemetryProvider creates a new telemetry provider with OTLP exporters
func NewTelemetryProvider(ctx context.Context, cfg config.OTelConfig) (*TelemetryProvider, error) {
	if cfg == nil || !cfg.Enabled() {
		return nil, nil
	}

	tp := &TelemetryProvider{
		shutdownFuncs: make([]func(context.Context) error, 0),
	}

	// Create resource with service information
	res, err := tp.createResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Setup OTLP connection options
	var dialOpts []grpc.DialOption
	if cfg.OTLPInsecure() {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Create trace exporter
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint()),
		otlptracegrpc.WithDialOption(dialOpts...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}
	tp.shutdownFuncs = append(tp.shutdownFuncs, traceExporter.Shutdown)

	// Create tracer provider with adaptive sampler
	sampler := NewAdaptiveSampler(cfg.TraceSampleRate())
	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(res),
		trace.WithSampler(sampler),
	)
	tp.tracerProvider = tracerProvider
	tp.shutdownFuncs = append(tp.shutdownFuncs, tracerProvider.Shutdown)

	// Set global tracer provider
	otel.SetTracerProvider(tracerProvider)

	// Create metric exporter
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint()),
		otlpmetricgrpc.WithDialOption(dialOpts...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}
	tp.shutdownFuncs = append(tp.shutdownFuncs, metricExporter.Shutdown)

	// Create meter provider with periodic reader
	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(
			metricExporter,
			metric.WithInterval(time.Duration(cfg.MetricsIntervalSeconds())*time.Second),
		)),
	)
	tp.meterProvider = meterProvider
	tp.shutdownFuncs = append(tp.shutdownFuncs, meterProvider.Shutdown)

	// Set global meter provider
	otel.SetMeterProvider(meterProvider)

	// Setup W3C Trace Context propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

// createResource creates an OpenTelemetry resource with service information
func (tp *TelemetryProvider) createResource(cfg config.OTelConfig) (*resource.Resource, error) {
	hostname, _ := os.Hostname()

	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName()),
		semconv.ServiceVersion(cfg.ServiceVersion()),
		semconv.DeploymentEnvironment(cfg.Environment()),
		semconv.HostName(hostname),
	), nil
}

// Shutdown gracefully shuts down all telemetry providers
func (tp *TelemetryProvider) Shutdown(ctx context.Context) error {
	var errs []error
	for _, fn := range tp.shutdownFuncs {
		if err := fn(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("telemetry shutdown errors: %v", errs)
	}
	return nil
}

// Tracer returns a tracer with the given name
func (tp *TelemetryProvider) Tracer(name string) otelTrace.Tracer {
	if tp == nil || tp.tracerProvider == nil {
		return otel.Tracer(name) // Returns no-op tracer if not initialized
	}
	return tp.tracerProvider.Tracer(name)
}

// Meter returns a meter with the given name
func (tp *TelemetryProvider) Meter(name string) otelMetric.Meter {
	if tp == nil || tp.meterProvider == nil {
		return otel.Meter(name) // Returns no-op meter if not initialized
	}
	return tp.meterProvider.Meter(name)
}
