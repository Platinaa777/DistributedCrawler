package worker

import (
	"context"
	"distributed-crawler/internal/domain/crawl/events"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/repos/outbox"
	"distributed-crawler/internal/infra/messaging/rabbitmq"
	"distributed-crawler/internal/infra/persistence"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

const (
	// DefaultOutboxPollInterval is the default interval for polling outbox events
	DefaultOutboxPollInterval = 10 * time.Second
	// DefaultOutboxBatchSize is the default number of events to fetch in one batch
	DefaultOutboxBatchSize = 100
)

// OutboxPublisher is a worker that polls outbox events and publishes them to RabbitMQ
type OutboxPublisher struct {
	outboxRepo   outbox.OutboxRepository
	txManager    persistence.TxManager
	rmqClient    rabbitmq.Client
	queueName    string
	pollInterval time.Duration
	batchSize    int
	logger       *zap.Logger
	stopChan     chan struct{}
}

// NewOutboxPublisher creates a new outbox publisher worker
func NewOutboxPublisher(
	outboxRepo outbox.OutboxRepository,
	txManager persistence.TxManager,
	rmqClient rabbitmq.Client,
	queueName string,
	logger *zap.Logger,
) *OutboxPublisher {
	return &OutboxPublisher{
		outboxRepo:   outboxRepo,
		txManager:    txManager,
		rmqClient:    rmqClient,
		queueName:    queueName,
		pollInterval: DefaultOutboxPollInterval,
		batchSize:    DefaultOutboxBatchSize,
		logger:       logger,
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
		events, err := w.outboxRepo.FetchUnprocessedEvents(ctxTX, w.batchSize)
		if err != nil {
			return fmt.Errorf("failed to fetch unprocessed events: %w", err)
		}

		if len(events) == 0 {
			w.logger.Debug("No unprocessed outbox events found")
			return nil
		}

		w.logger.Info("Processing outbox events", zap.Int("count", len(events)))

		// Process each event
		for _, event := range events {
			if err := w.processEvent(ctxTX, event); err != nil {
				w.logger.Error("Failed to process outbox event",
					zap.String("event_id", event.ID.String()),
					zap.String("event_type", event.EventType),
					zap.Error(err),
				)
				// Continue processing other events even if one fails
				continue
			}
		}

		return nil
	})

	if err != nil {
		w.logger.Error("Failed to process outbox events", zap.Error(err))
	}
}

// processEvent publishes a single outbox event to RabbitMQ and marks it as processed
func (w *OutboxPublisher) processEvent(ctx context.Context, event *models.OutboxEvent) error {
	// Parse event based on type
	switch event.EventType {
	case string(events.EventTypeTaskEnqueued):
		return w.publishTaskEnqueuedEvent(ctx, event)
	default:
		w.logger.Warn("Unknown event type", zap.String("event_type", event.EventType))
		// Mark as processed anyway to avoid reprocessing
		return w.outboxRepo.MarkAsProcessed(ctx, event.ID)
	}
}

// publishTaskEnqueuedEvent publishes a TaskEnqueuedEvent to RabbitMQ
func (w *OutboxPublisher) publishTaskEnqueuedEvent(ctx context.Context, outboxEvent *models.OutboxEvent) error {
	// Unmarshal payload
	var event events.TaskEnqueuedEvent
	if err := json.Unmarshal(outboxEvent.Payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event payload: %w", err)
	}

	// Create message for RabbitMQ (use the same structure as before)
	message := rabbitmq.CrawlTaskMessage{
		TaskID:     event.TaskID,
		JobID:      event.JobID,
		URL:        event.URL,
		EnqueuedAt: event.EnqueuedAt,
	}

	// Publish to RabbitMQ
	if err := w.rmqClient.Publish(ctx, w.queueName, message); err != nil {
		return fmt.Errorf("failed to publish task to RabbitMQ: %w", err)
	}

	w.logger.Debug("Published task to RabbitMQ",
		zap.String("task_id", event.TaskID),
		zap.String("url", event.URL),
	)

	// Mark event as processed
	if err := w.outboxRepo.MarkAsProcessed(ctx, outboxEvent.ID); err != nil {
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
