package converters

import (
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"
)

func SaveOutboxEventToSnapshot(event models.OutboxEvent) snapshots.OutboxEventSnapshot {
	return snapshots.OutboxEventSnapshot{
		ID:          event.ID.String(),
		EventType:   event.EventType,
		AggregateID: event.AggregateID,
		Payload:     event.Payload,
		OccurredAt:  event.OccurredAt,
		ProcessedAt: event.ProcessedAt,
		CreatedAt:   event.CreatedAt,
	}
}

func RestoreOutboxEventFromSnapshot(snapshot snapshots.OutboxEventSnapshot) (*models.OutboxEvent, error) {
	id, err := valueobjects.NewOutboxEventID(snapshot.ID)
	if err != nil {
		return nil, err
	}

	return &models.OutboxEvent{
		ID:          id,
		EventType:   snapshot.EventType,
		AggregateID: snapshot.AggregateID,
		Payload:     snapshot.Payload,
		OccurredAt:  snapshot.OccurredAt,
		ProcessedAt: snapshot.ProcessedAt,
		CreatedAt:   snapshot.CreatedAt,
	}, nil
}
