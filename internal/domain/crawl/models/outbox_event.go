package models

import (
	"time"

	"distributed-crawler/internal/domain/crawl/valueobjects"
)

// OutboxEvent represents a domain event stored in the outbox table
type OutboxEvent struct {
	ID          valueobjects.OutboxEventID
	EventType   string
	AggregateID string
	Payload     []byte
	OccurredAt  time.Time
	ProcessedAt *time.Time
	CreatedAt   time.Time
}

// IsProcessed returns true if the event has been processed
func (e *OutboxEvent) IsProcessed() bool {
	return e.ProcessedAt != nil
}

// MarkAsProcessed marks the event as processed
func (e *OutboxEvent) MarkAsProcessed() {
	now := time.Now().UTC()
	e.ProcessedAt = &now
}
