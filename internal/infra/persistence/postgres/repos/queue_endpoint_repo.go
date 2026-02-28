package repos

import (
	"context"
	"database/sql"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"

	queuemodels "distributed-crawler/internal/domain/queue/models"
	queuerepos "distributed-crawler/internal/domain/queue/repos"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/infra/persistence/postgres/converters"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"
)

const (
	queueEndpointsTable = "queue_endpoints"
	queueRulesTable     = "queue_routing_rules"

	// queue_endpoints columns
	colQEID          = "id"
	colQEDisplayName = "display_name"
	colQEBrokerType  = "broker_type"
	colQEStage       = "stage"
	colQEHost        = "host"
	colQEQueueName = "queue_name"
	colQESecretKey = "secret_key"
	colQECreatedAt = "created_at"
	colQEUpdatedAt   = "updated_at"

	// queue_routing_rules columns
	colRuleID    = "id"
	colRuleStage = "stage"
	colRuleScope = "scope"
)

// -- QueueEndpointRepository --

type queueEndpointRepository struct {
	client persistence.Client
}

// NewQueueEndpointRepository creates a new postgres-backed QueueEndpointRepository.
func NewQueueEndpointRepository(client persistence.Client) queuerepos.QueueEndpointRepository {
	return &queueEndpointRepository{client: client}
}

func (r *queueEndpointRepository) Create(ctx context.Context, endpoint queuemodels.QueueEndpoint) (*queuemodels.QueueEndpoint, error) {
	now := time.Now().UTC()
	builder := sq.Insert(queueEndpointsTable).
		PlaceholderFormat(sq.Dollar).
		Columns(
			colQEDisplayName, colQEBrokerType, colQEStage, colQEHost,
			colQEQueueName, colQESecretKey, colQECreatedAt, colQEUpdatedAt,
		).
		Values(
			endpoint.DisplayName, endpoint.BrokerType, endpoint.Stage, endpoint.Host,
			endpoint.QueueName, endpoint.SecretKey, now, now,
		).
		Suffix("RETURNING id, display_name, broker_type, stage, host, queue_name, secret_key, created_at, updated_at")

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{Name: "queue_endpoint_repo.Create", QueryRaw: query}
	var s snapshots.QueueEndpointSnapshot
	row := r.client.DB().QueryRowContext(ctx, q, args...)
	if err := scanQueueEndpoint(row, &s); err != nil {
		return nil, err
	}

	return converters.RestoreQueueEndpointFromSnapshot(s), nil
}

func (r *queueEndpointRepository) Get(ctx context.Context, id string) (*queuemodels.QueueEndpoint, error) {
	builder := sq.Select(
		colQEID, colQEDisplayName, colQEBrokerType, colQEStage, colQEHost,
		colQEQueueName, colQESecretKey, colQECreatedAt, colQEUpdatedAt,
	).
		PlaceholderFormat(sq.Dollar).
		From(queueEndpointsTable).
		Where(sq.Eq{"id": id}).
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{Name: "queue_endpoint_repo.Get", QueryRaw: query}
	var s snapshots.QueueEndpointSnapshot
	row := r.client.DB().QueryRowContext(ctx, q, args...)
	if err := scanQueueEndpoint(row, &s); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return converters.RestoreQueueEndpointFromSnapshot(s), nil
}

func (r *queueEndpointRepository) List(ctx context.Context) ([]*queuemodels.QueueEndpoint, error) {
	builder := sq.Select(
		colQEID, colQEDisplayName, colQEBrokerType, colQEStage, colQEHost,
		colQEQueueName, colQESecretKey, colQECreatedAt, colQEUpdatedAt,
	).
		PlaceholderFormat(sq.Dollar).
		From(queueEndpointsTable).
		OrderBy(colQECreatedAt + " ASC")

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{Name: "queue_endpoint_repo.List", QueryRaw: query}
	rows, err := r.client.DB().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*queuemodels.QueueEndpoint
	for rows.Next() {
		var s snapshots.QueueEndpointSnapshot
		if err := scanQueueEndpoint(rows, &s); err != nil {
			return nil, err
		}
		result = append(result, converters.RestoreQueueEndpointFromSnapshot(s))
	}

	return result, rows.Err()
}

func (r *queueEndpointRepository) Update(ctx context.Context, endpoint queuemodels.QueueEndpoint) (*queuemodels.QueueEndpoint, error) {
	now := time.Now().UTC()
	builder := sq.Update(queueEndpointsTable).
		PlaceholderFormat(sq.Dollar).
		Set(colQEDisplayName, endpoint.DisplayName).
		Set(colQEBrokerType, endpoint.BrokerType).
		Set(colQEStage, endpoint.Stage).
		Set(colQEHost, endpoint.Host).
		Set(colQEQueueName, endpoint.QueueName).
		Set(colQESecretKey, endpoint.SecretKey).
		Set(colQEUpdatedAt, now).
		Where(sq.Eq{colQEID: endpoint.ID}).
		Suffix("RETURNING id, display_name, broker_type, stage, host, queue_name, secret_key, created_at, updated_at")

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{Name: "queue_endpoint_repo.Update", QueryRaw: query}
	var s snapshots.QueueEndpointSnapshot
	row := r.client.DB().QueryRowContext(ctx, q, args...)
	if err := scanQueueEndpoint(row, &s); err != nil {
		return nil, err
	}

	return converters.RestoreQueueEndpointFromSnapshot(s), nil
}

func (r *queueEndpointRepository) Delete(ctx context.Context, id string) error {
	builder := sq.Delete(queueEndpointsTable).
		PlaceholderFormat(sq.Dollar).
		Where(sq.Eq{colQEID: id})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := persistence.Query{Name: "queue_endpoint_repo.Delete", QueryRaw: query}
	_, err = r.client.DB().ExecContext(ctx, q, args...)
	return err
}

// scanQueueEndpoint scans a row into a QueueEndpointSnapshot.
type queueEndpointScanner interface {
	Scan(dest ...interface{}) error
}

func scanQueueEndpoint(row queueEndpointScanner, s *snapshots.QueueEndpointSnapshot) error {
	return row.Scan(
		&s.ID, &s.DisplayName, &s.BrokerType, &s.Stage, &s.Host,
		&s.QueueName, &s.SecretKey, &s.CreatedAt, &s.UpdatedAt,
	)
}

// -- QueueRoutingRuleRepository --

type queueRoutingRuleRepository struct {
	client persistence.Client
}

// NewQueueRoutingRuleRepository creates a new postgres-backed QueueRoutingRuleRepository.
func NewQueueRoutingRuleRepository(client persistence.Client) queuerepos.QueueRoutingRuleRepository {
	return &queueRoutingRuleRepository{client: client}
}

// Upsert inserts or replaces a routing rule (by stage+scope).
func (r *queueRoutingRuleRepository) Upsert(ctx context.Context, rule queuemodels.RoutingRule) (*queuemodels.RoutingRule, error) {
	now := time.Now().UTC()

	upsertRule := sq.Insert(queueRulesTable).
		PlaceholderFormat(sq.Dollar).
		Columns(colRuleStage, colRuleScope, colQECreatedAt, colQEUpdatedAt).
		Values(rule.Stage, rule.Scope, now, now).
		Suffix("ON CONFLICT (stage, scope) DO UPDATE SET updated_at = EXCLUDED.updated_at RETURNING id, stage, scope")

	ruleSQL, ruleArgs, err := upsertRule.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{Name: "queue_routing_rule_repo.Upsert", QueryRaw: ruleSQL}
	var ruleSnap snapshots.QueueRoutingRuleSnapshot
	row := r.client.DB().QueryRowContext(ctx, q, ruleArgs...)
	if err := row.Scan(&ruleSnap.ID, &ruleSnap.Stage, &ruleSnap.Scope); err != nil {
		return nil, err
	}

	return converters.RestoreRoutingRuleFromSnapshot(ruleSnap), nil
}

func (r *queueRoutingRuleRepository) ListByStage(ctx context.Context, stage queuemodels.Stage) ([]*queuemodels.RoutingRule, error) {
	rulesBuilder := sq.Select(colRuleID, colRuleStage, colRuleScope).
		PlaceholderFormat(sq.Dollar).
		From(queueRulesTable)
	if stage != "" {
		rulesBuilder = rulesBuilder.Where(sq.Eq{colRuleStage: stage})
	}

	rulesSQL, rulesArgs, err := rulesBuilder.ToSql()
	if err != nil {
		return nil, err
	}

	rulesQ := persistence.Query{Name: "queue_routing_rule_repo.ListByStage", QueryRaw: rulesSQL}
	ruleRows, err := r.client.DB().QueryContext(ctx, rulesQ, rulesArgs...)
	if err != nil {
		return nil, err
	}
	defer ruleRows.Close()

	var rules []*queuemodels.RoutingRule
	for ruleRows.Next() {
		var rs snapshots.QueueRoutingRuleSnapshot
		if err := ruleRows.Scan(&rs.ID, &rs.Stage, &rs.Scope); err != nil {
			return nil, err
		}
		rules = append(rules, converters.RestoreRoutingRuleFromSnapshot(rs))
	}

	return rules, ruleRows.Err()
}
