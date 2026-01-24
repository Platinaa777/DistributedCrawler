package events

import (
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of domain event
type EventType string

const (
	EventTypeTaskEnqueued EventType = "task.enqueued"
)

// BaseEvent contains common fields for all domain events
type BaseEvent struct {
	ID         string    `json:"id"`
	Type       EventType `json:"type"`
	OccurredAt time.Time `json:"occurred_at"`
}

// NewBaseEvent creates a new base event with generated ID and current timestamp
func NewBaseEvent(eventType EventType) BaseEvent {
	return BaseEvent{
		ID:         uuid.New().String(),
		Type:       eventType,
		OccurredAt: time.Now().UTC(),
	}
}
