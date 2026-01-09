package repos

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/repos/outbox"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/infra/persistence/postgres/converters"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"

	sq "github.com/Masterminds/squirrel"
)

const (
	outboxTableName = "crawl_task_outbox"

	outboxIDColumn          = "id"
	outboxEventTypeColumn   = "event_type"
	outboxAggregateIDColumn = "aggregate_id"
	outboxPayloadColumn     = "payload"
	outboxOccurredAtColumn  = "occurred_at"
	outboxProcessedAtColumn = "processed_at"
	outboxCreatedAtColumn   = "created_at"
)

type outboxRepository struct {
	client persistence.Client
}

func NewOutboxRepository(client persistence.Client) outbox.OutboxRepository {
	return &outboxRepository{client: client}
}

func (r *outboxRepository) Create(ctx context.Context, event models.OutboxEvent) error {
	dbEntity := converters.SaveOutboxEventToSnapshot(event)

	builder := sq.Insert(outboxTableName).
		PlaceholderFormat(sq.Dollar).
		Columns(
			outboxIDColumn,
			outboxEventTypeColumn,
			outboxAggregateIDColumn,
			outboxPayloadColumn,
			outboxOccurredAtColumn,
			outboxProcessedAtColumn,
			outboxCreatedAtColumn,
		).
		Values(
			dbEntity.ID,
			dbEntity.EventType,
			dbEntity.AggregateID,
			dbEntity.Payload,
			dbEntity.OccurredAt,
			dbEntity.ProcessedAt,
			dbEntity.CreatedAt,
		)

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := persistence.Query{
		Name:     "outbox_repository.Create",
		QueryRaw: query,
	}

	_, err = r.client.DB().ExecContext(ctx, q, args...)
	return err
}

func (r *outboxRepository) FetchUnprocessedEvents(ctx context.Context, limit int) ([]*models.OutboxEvent, error) {
	builder := sq.Select(
		outboxIDColumn,
		outboxEventTypeColumn,
		outboxAggregateIDColumn,
		outboxPayloadColumn,
		outboxOccurredAtColumn,
		outboxProcessedAtColumn,
		outboxCreatedAtColumn,
	).
		PlaceholderFormat(sq.Dollar).
		From(outboxTableName).
		Where(sq.Eq{outboxProcessedAtColumn: nil}).
		OrderBy(outboxOccurredAtColumn + " ASC").
		Limit(uint64(limit)).
		Suffix("FOR UPDATE SKIP LOCKED")

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "outbox_repository.FetchUnprocessedEvents",
		QueryRaw: query,
	}

	var snapshots []snapshots.OutboxEventSnapshot
	err = r.client.DB().ScanAllContext(ctx, &snapshots, q, args...)
	if err != nil {
		return nil, err
	}

	events := make([]*models.OutboxEvent, 0, len(snapshots))
	for _, snapshot := range snapshots {
		event, err := converters.RestoreOutboxEventFromSnapshot(snapshot)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
}

func (r *outboxRepository) MarkAsProcessed(ctx context.Context, id valueobjects.OutboxEventID) error {
	builder := sq.Update(outboxTableName).
		PlaceholderFormat(sq.Dollar).
		Set(outboxProcessedAtColumn, sq.Expr("NOW()")).
		Where(sq.Eq{outboxIDColumn: id.String()})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := persistence.Query{
		Name:     "outbox_repository.MarkAsProcessed",
		QueryRaw: query,
	}

	_, err = r.client.DB().ExecContext(ctx, q, args...)
	return err
}
