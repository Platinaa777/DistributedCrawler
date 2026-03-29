package worker

import (
	"context"
	"distributed-crawler/internal/domain/crawl/events"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/repos/outbox"
	"distributed-crawler/internal/infra/messaging"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/telemetry"
	"encoding/json"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	// DefaultOutboxPollInterval is the default interval for polling outbox events
	DefaultOutboxPollInterval = 10 * time.Second
	// DefaultOutboxBatchSize is the default number of events to fetch in one batch
	DefaultOutboxBatchSize = 100
)

// OutboxPublisher is a worker that polls outbox events and publishes them to the message broker
type OutboxPublisher struct {
	outboxRepo   outbox.OutboxRepository
	txManager    persistence.TxManager
	msgClient    messaging.Client
	queueName    string
	pollInterval time.Duration
	batchSize    int
	logger       *zap.Logger
	tracer       trace.Tracer
	stopChan     chan struct{}
}

// NewOutboxPublisher creates a new outbox publisher worker.
func NewOutboxPublisher(
	outboxRepo outbox.OutboxRepository,
	txManager persistence.TxManager,
	msgClient messaging.Client,
	queueName string,
	logger *zap.Logger,
	tracer trace.Tracer,
) *OutboxPublisher {
	return &OutboxPublisher{
		outboxRepo:   outboxRepo,
		txManager:    txManager,
		msgClient:    msgClient,
		queueName:    queueName,
		pollInterval: DefaultOutboxPollInterval,
		batchSize:    DefaultOutboxBatchSize,
		logger:       logger,
		tracer:       tracer,
		stopChan:     make(chan struct{}),
	}
}

// Start starts the worker in a goroutine
func (w *OutboxPublisher) Start(ctx context.Context) {
	w.logger.Info("Starting outbox publisher worker",
		zap.Duration("poll_interval", w.pollInterval),
		zap.Int("batch_size", w.batchSize),
	)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Run immediately on start
	w.processOutboxEvents(ctx)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Outbox publisher worker stopped due to context cancellation")
			return
		case <-w.stopChan:
			w.logger.Info("Outbox publisher worker stopped")
			return
		case <-ticker.C:
			w.processOutboxEvents(ctx)
		}
	}
}

// Stop stops the worker
func (w *OutboxPublisher) Stop() {
	close(w.stopChan)
}

// processOutboxEvents fetches unprocessed outbox events and publishes them to RabbitMQ
func (w *OutboxPublisher) processOutboxEvents(ctx context.Context) {
	err := w.txManager.ReadCommitted(ctx, func(ctxTX context.Context) error {
		// Fetch unprocessed events with row-level locking
		evts, err := w.outboxRepo.FetchUnprocessedEvents(ctxTX, w.batchSize)
		if err != nil {
			return fmt.Errorf("failed to fetch unprocessed events: %w", err)
		}

		if len(evts) == 0 {
			w.logger.Debug("No unprocessed outbox events found")
			return nil
		}

		w.logger.Info("Processing outbox events", zap.Int("count", len(evts)))

		// Process each event. ctx (no-tx) is passed for network/routing calls;
		// ctxTX is used only for DB writes inside processEvent.
		for _, event := range evts {
			if err := w.processEvent(ctx, ctxTX, event); err != nil {
				// Return on the first error to avoid running further DB ops
				// on an already-aborted transaction.
				return fmt.Errorf("failed to process outbox event %s: %w", event.ID, err)
			}
		}

		return nil
	})

	if err != nil {
		w.logger.Error("Failed to process outbox events", zap.Error(err))
	}
}

// processEvent publishes a single outbox event to the message broker and marks it as processed.
// ctx is the original context (no DB transaction) used for routing and network calls.
// ctxTX is the transaction context used only for DB writes.
func (w *OutboxPublisher) processEvent(ctx, ctxTX context.Context, event *models.OutboxEvent) error {
	switch event.EventType {
	case string(events.EventTypeTaskEnqueued):
		return w.publishTaskEnqueuedEvent(ctx, ctxTX, event)
	default:
		w.logger.Warn("Unknown event type", zap.String("event_type", event.EventType))
		return w.outboxRepo.MarkAsProcessed(ctxTX, event.ID)
	}
}

// publishTaskEnqueuedEvent publishes a TaskEnqueuedEvent to the message broker.
// ctx (no-tx) is used for routing lookups and the broker publish.
// ctxTX is used only for the final DB write (MarkAsProcessed).
func (w *OutboxPublisher) publishTaskEnqueuedEvent(ctx, ctxTX context.Context, outboxEvent *models.OutboxEvent) error {
	var event events.TaskEnqueuedEvent
	if err := json.Unmarshal(outboxEvent.Payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event payload: %w", err)
	}

	// Restore the trace context that was captured when the task was originally created
	// (e.g. the gRPC request span or the parser span that discovered this link).
	// This makes the outbox publish span a child of that original trace instead of
	// starting a brand-new, disconnected trace.
	if len(event.TraceContext) > 0 {
		ctx = telemetry.ExtractTraceContext(ctx, event.TraceContext)
	}

	targetQueue := w.queueName
	if event.TargetQueue != "" {
		targetQueue = event.TargetQueue
	}

	// Start a child span — now correctly linked to the originating trace
	var traceCtx map[string]string
	if w.tracer != nil {
		var span trace.Span
		ctx, span = w.tracer.Start(ctx, "outbox_publish_task",
			trace.WithSpanKind(trace.SpanKindProducer),
			trace.WithAttributes(
				attribute.String("task.id", event.TaskID),
				attribute.String("job.id", event.JobID),
				attribute.String("task.url", event.URL),
				attribute.String("messaging.destination", targetQueue),
			),
		)
		defer span.End()

		traceCtx = telemetry.InjectTraceContext(ctx)
	}

	message := messaging.CrawlTaskMessage{
		TaskID:       event.TaskID,
		JobID:        event.JobID,
		URL:          event.URL,
		EnqueuedAt:   event.EnqueuedAt,
		TraceContext: traceCtx,
	}

	if err := w.msgClient.Publish(ctx, targetQueue, message); err != nil {
		return fmt.Errorf("failed to publish task to message broker: %w", err)
	}

	w.logger.Debug("Published task to message broker",
		zap.String("task_id", event.TaskID),
		zap.String("url", event.URL),
		zap.String("queue", targetQueue),
	)

	// Mark event as processed inside the outbox transaction.
	if err := w.outboxRepo.MarkAsProcessed(ctxTX, outboxEvent.ID); err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}

	w.logger.Info("Outbox event processed successfully",
		zap.String("event_id", outboxEvent.ID.String()),
		zap.String("task_id", event.TaskID),
		zap.String("job_id", event.JobID),
		zap.String("url", event.URL),
	)

	return nil
}
