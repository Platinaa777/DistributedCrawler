package telemetry

import (
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds all OpenTelemetry metric instruments for the application
type Metrics struct {
	// API Metrics
	HTTPRequestsTotal   metric.Int64Counter
	HTTPRequestDuration metric.Float64Histogram

	// Task Lifecycle Metrics
	TasksCreatedTotal   metric.Int64Counter
	TasksCompletedTotal metric.Int64Counter
	TasksFailedTotal    metric.Int64Counter

	// Worker Performance Metrics
	FetchDuration    metric.Float64Histogram
	FetchErrorsTotal metric.Int64Counter
	ParserDuration   metric.Float64Histogram

	// Infrastructure Metrics
	OutboxEventsPending   metric.Int64Gauge
	WorkerHeartbeatsTotal metric.Int64Counter
	WorkerActiveCount     metric.Int64Gauge
}

// NewMetrics creates all metric instruments
func NewMetrics(meter metric.Meter) (*Metrics, error) {
	m := &Metrics{}
	var err error

	// API Metrics
	m.HTTPRequestsTotal, err = meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	m.HTTPRequestDuration, err = meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10),
	)
	if err != nil {
		return nil, err
	}

	// Task Lifecycle Metrics
	m.TasksCreatedTotal, err = meter.Int64Counter(
		"crawl_tasks_created_total",
		metric.WithDescription("Total number of crawl tasks created"),
		metric.WithUnit("{task}"),
	)
	if err != nil {
		return nil, err
	}

	m.TasksCompletedTotal, err = meter.Int64Counter(
		"crawl_tasks_completed_total",
		metric.WithDescription("Total number of crawl tasks completed successfully"),
		metric.WithUnit("{task}"),
	)
	if err != nil {
		return nil, err
	}

	m.TasksFailedTotal, err = meter.Int64Counter(
		"crawl_tasks_failed_total",
		metric.WithDescription("Total number of crawl tasks that failed"),
		metric.WithUnit("{task}"),
	)
	if err != nil {
		return nil, err
	}

	// Worker Performance Metrics
	m.FetchDuration, err = meter.Float64Histogram(
		"fetch_request_duration_seconds",
		metric.WithDescription("Duration of fetch requests in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60),
	)
	if err != nil {
		return nil, err
	}

	m.FetchErrorsTotal, err = meter.Int64Counter(
		"fetch_errors_total",
		metric.WithDescription("Total number of fetch errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, err
	}

	m.ParserDuration, err = meter.Float64Histogram(
		"parser_duration_seconds",
		metric.WithDescription("Duration of parsing operations in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5),
	)
	if err != nil {
		return nil, err
	}

	// Infrastructure Metrics
	m.OutboxEventsPending, err = meter.Int64Gauge(
		"outbox_events_pending",
		metric.WithDescription("Number of pending outbox events"),
		metric.WithUnit("{event}"),
	)
	if err != nil {
		return nil, err
	}

	m.WorkerHeartbeatsTotal, err = meter.Int64Counter(
		"worker_heartbeats_total",
		metric.WithDescription("Total number of worker heartbeats"),
		metric.WithUnit("{heartbeat}"),
	)
	if err != nil {
		return nil, err
	}

	m.WorkerActiveCount, err = meter.Int64Gauge(
		"worker_active_count",
		metric.WithDescription("Number of active workers"),
		metric.WithUnit("{worker}"),
	)
	if err != nil {
		return nil, err
	}

	return m, nil
}
