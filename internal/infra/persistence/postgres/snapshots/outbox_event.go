package snapshots

import "time"

type OutboxEventSnapshot struct {
	ID          string     `db:"id"`
	EventType   string     `db:"event_type"`
	AggregateID string     `db:"aggregate_id"`
	Payload     []byte     `db:"payload"`
	OccurredAt  time.Time  `db:"occurred_at"`
	ProcessedAt *time.Time `db:"processed_at"`
	CreatedAt   time.Time  `db:"created_at"`
}
