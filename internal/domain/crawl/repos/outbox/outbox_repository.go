package outbox

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
)

type OutboxRepository interface {
	// Create stores a new outbox event
	Create(ctx context.Context, event models.OutboxEvent) error

	// FetchUnprocessedEvents fetches unprocessed events with row-level locking
	// Uses SELECT FOR UPDATE SKIP LOCKED for concurrent worker safety
	FetchUnprocessedEvents(ctx context.Context, limit int) ([]*models.OutboxEvent, error)

	// MarkAsProcessed marks an event as processed
	MarkAsProcessed(ctx context.Context, id valueobjects.OutboxEventID) error
}
