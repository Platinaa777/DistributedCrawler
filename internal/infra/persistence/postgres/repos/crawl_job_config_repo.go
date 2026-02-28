package repos

import (
	"context"
	"fmt"
	"strings"

	"distributed-crawler/internal/domain/crawl/models"
	crawljobconfig "distributed-crawler/internal/domain/crawl/repos/crawl_job_config"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/infra/persistence/postgres/converters"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"

	sq "github.com/Masterminds/squirrel"
)

const (
	configTableName             = "crawl_job_configs"
	configQueueEndpointsTable   = "crawl_job_config_queue_endpoints"

	configIDColumn               = "id"
	configNameColumn             = "name"
	configExtractionSpecColumn   = "extraction_spec"
	configScopesColumn           = "scopes"
	configSeedsColumn            = "seeds"
	configRateLimitColumn        = "rate_limit"
	configRetriesColumn          = "retries"
	configAuthColumn             = "auth"
	configScheduleColumn         = "schedule"
	configJobTypeColumn          = "job_type"
	configRespectRobotsTxtColumn = "respect_robots_txt"
	configCrawlModeColumn        = "crawl_mode"
)

type crawlJobConfigRepository struct {
	client persistence.Client
}

func NewCrawlJobConfigRepository(client persistence.Client) crawljobconfig.CrawlJobConfigRepository {
	return &crawlJobConfigRepository{client: client}
}

func (r *crawlJobConfigRepository) Create(ctx context.Context, entity models.CrawlJobConfig) (valueobjects.ID, error) {
	dbEntity, err := converters.SaveCrawlJobConfigToSnapshot(entity)
	if err != nil {
		return valueobjects.ID{}, err
	}

	builder := sq.Insert(configTableName).
		PlaceholderFormat(sq.Dollar).
		Columns(
			configIDColumn,
			configNameColumn,
			configExtractionSpecColumn,
			configScopesColumn,
			configSeedsColumn,
			configRateLimitColumn,
			configRetriesColumn,
			configAuthColumn,
			configScheduleColumn,
			configJobTypeColumn,
			configRespectRobotsTxtColumn,
			configCrawlModeColumn,
		).
		Values(
			dbEntity.ID,
			dbEntity.Name,
			dbEntity.ExtractionSpec,
			dbEntity.Scopes,
			dbEntity.Seeds,
			dbEntity.RateLimit,
			dbEntity.Retries,
			dbEntity.Auth,
			dbEntity.Schedule,
			dbEntity.JobType,
			dbEntity.RespectRobotsTxt,
			dbEntity.CrawlMode,
		).
		Suffix("RETURNING id")

	query, args, err := builder.ToSql()
	if err != nil {
		return valueobjects.ID{}, err
	}

	q := persistence.Query{
		Name:     "crawl_job_config_repository.Create",
		QueryRaw: query,
	}

	var id string
	err = r.client.DB().QueryRowContext(ctx, q, args...).Scan(&id)
	if err != nil {
		return valueobjects.ID{}, err
	}

	if err := r.insertQueueEndpoints(ctx, id, entity.QueueEndpointAssignments); err != nil {
		return valueobjects.ID{}, err
	}

	return valueobjects.NewID(id)
}

func (r *crawlJobConfigRepository) Get(ctx context.Context, id valueobjects.ID) (*models.CrawlJobConfig, error) {
	builder := sq.Select(
		configIDColumn,
		configNameColumn,
		configExtractionSpecColumn,
		configScopesColumn,
		configSeedsColumn,
		configRateLimitColumn,
		configRetriesColumn,
		configAuthColumn,
		configScheduleColumn,
		configJobTypeColumn,
		configRespectRobotsTxtColumn,
		configCrawlModeColumn,
	).
		PlaceholderFormat(sq.Dollar).
		From(configTableName).
		Where(sq.Eq{configIDColumn: id.String()}).
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "crawl_job_config_repository.Get",
		QueryRaw: query,
	}

	var snapshot snapshots.CrawlJobConfigSnapshot
	err = r.client.DB().ScanOneContext(ctx, &snapshot, q, args...)
	if err != nil {
		return nil, err
	}

	assignments, err := fetchQueueEndpointAssignments(ctx, r.client, snapshot.ID)
	if err != nil {
		return nil, err
	}
	snapshot.QueueEndpointAssignments = assignments

	return converters.RestoreCrawlJobConfigFromSnapshot(snapshot)
}

func (r *crawlJobConfigRepository) Update(ctx context.Context, entity models.CrawlJobConfig) error {
	dbEntity, err := converters.SaveCrawlJobConfigToSnapshot(entity)
	if err != nil {
		return err
	}

	builder := sq.Update(configTableName).
		PlaceholderFormat(sq.Dollar).
		Set(configNameColumn, dbEntity.Name).
		Set(configExtractionSpecColumn, dbEntity.ExtractionSpec).
		Set(configScopesColumn, dbEntity.Scopes).
		Set(configSeedsColumn, dbEntity.Seeds).
		Set(configRateLimitColumn, dbEntity.RateLimit).
		Set(configRetriesColumn, dbEntity.Retries).
		Set(configAuthColumn, dbEntity.Auth).
		Set(configScheduleColumn, dbEntity.Schedule).
		Set(configJobTypeColumn, dbEntity.JobType).
		Set(configRespectRobotsTxtColumn, dbEntity.RespectRobotsTxt).
		Set(configCrawlModeColumn, dbEntity.CrawlMode).
		Where(sq.Eq{configIDColumn: dbEntity.ID})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := persistence.Query{
		Name:     "crawl_job_config_repository.Update",
		QueryRaw: query,
	}

	_, err = r.client.DB().ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}

	if err := r.replaceQueueEndpoints(ctx, dbEntity.ID, entity.QueueEndpointAssignments); err != nil {
		return err
	}

	return nil
}

func (r *crawlJobConfigRepository) Delete(ctx context.Context, id valueobjects.ID) error {
	builder := sq.Delete(configTableName).
		PlaceholderFormat(sq.Dollar).
		Where(sq.Eq{configIDColumn: id.String()})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := persistence.Query{
		Name:     "crawl_job_config_repository.Delete",
		QueryRaw: query,
	}

	_, err = r.client.DB().ExecContext(ctx, q, args...)
	return err
}

func (r *crawlJobConfigRepository) ListAllScheduled(ctx context.Context, limit, offset int) ([]*models.CrawlJobConfig, error) {
	builder := sq.Select(
		configIDColumn,
		configNameColumn,
		configExtractionSpecColumn,
		configScopesColumn,
		configSeedsColumn,
		configRateLimitColumn,
		configRetriesColumn,
		configAuthColumn,
		configScheduleColumn,
		configJobTypeColumn,
		configRespectRobotsTxtColumn,
		configCrawlModeColumn,
	).
		PlaceholderFormat(sq.Dollar).
		From(configTableName).
		Where(sq.Eq{configJobTypeColumn: models.JobTypeScheduled}).
		OrderBy(configNameColumn + " ASC").
		Limit(uint64(limit)).
		Offset(uint64(offset))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "crawl_job_config_repository.ListAll",
		QueryRaw: query,
	}

	var snaps []snapshots.CrawlJobConfigSnapshot
	err = r.client.DB().ScanAllContext(ctx, &snaps, q, args...)
	if err != nil {
		return nil, err
	}

	// Collect IDs for batch fetch of queue endpoints.
	configIDs := make([]string, len(snaps))
	for i, s := range snaps {
		configIDs[i] = s.ID
	}
	assignmentsByConfig, err := fetchQueueEndpointAssignmentsForMany(ctx, r.client, configIDs)
	if err != nil {
		return nil, err
	}

	configs := make([]*models.CrawlJobConfig, 0, len(snaps))
	for _, snapshot := range snaps {
		snapshot.QueueEndpointAssignments = assignmentsByConfig[snapshot.ID]
		config, err := converters.RestoreCrawlJobConfigFromSnapshot(snapshot)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	return configs, nil
}

// fetchQueueEndpointAssignments returns assignments for a single config.
func fetchQueueEndpointAssignments(ctx context.Context, client persistence.Client, configID string) ([]snapshots.QueueEndpointAssignmentSnap, error) {
	rawQuery := fmt.Sprintf(
		"SELECT queue_endpoint_id, weight FROM %s WHERE crawl_job_config_id = $1",
		configQueueEndpointsTable,
	)

	rows, err := client.DB().QueryContext(ctx, persistence.Query{
		Name:     "crawl_job_config_repository.fetchQueueEndpointAssignments",
		QueryRaw: rawQuery,
	}, configID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []snapshots.QueueEndpointAssignmentSnap
	for rows.Next() {
		var a snapshots.QueueEndpointAssignmentSnap
		if err := rows.Scan(&a.EndpointID, &a.Weight); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

// fetchQueueEndpointAssignmentsForMany returns a map of configID → assignments for a batch of configs.
func fetchQueueEndpointAssignmentsForMany(ctx context.Context, client persistence.Client, configIDs []string) (map[string][]snapshots.QueueEndpointAssignmentSnap, error) {
	result := make(map[string][]snapshots.QueueEndpointAssignmentSnap, len(configIDs))
	if len(configIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(configIDs))
	args := make([]any, len(configIDs))
	for i, id := range configIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	rawQuery := fmt.Sprintf(
		"SELECT crawl_job_config_id, queue_endpoint_id, weight FROM %s WHERE crawl_job_config_id IN (%s)",
		configQueueEndpointsTable,
		strings.Join(placeholders, ", "),
	)

	rows, err := client.DB().QueryContext(ctx, persistence.Query{
		Name:     "crawl_job_config_repository.fetchQueueEndpointAssignmentsForMany",
		QueryRaw: rawQuery,
	}, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var configID string
		var a snapshots.QueueEndpointAssignmentSnap
		if err := rows.Scan(&configID, &a.EndpointID, &a.Weight); err != nil {
			return nil, err
		}
		result[configID] = append(result[configID], a)
	}
	return result, rows.Err()
}

// insertQueueEndpoints inserts rows into the join table for the given configID.
func (r *crawlJobConfigRepository) insertQueueEndpoints(ctx context.Context, configID string, assignments []models.QueueEndpointAssignment) error {
	if len(assignments) == 0 {
		return nil
	}

	placeholders := make([]string, len(assignments))
	args := make([]any, 0, len(assignments)*3)
	for i, a := range assignments {
		placeholders[i] = fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3)
		w := a.Weight
		if w <= 0 {
			w = 1
		}
		args = append(args, configID, a.EndpointID, w)
	}

	rawQuery := fmt.Sprintf(
		"INSERT INTO %s (crawl_job_config_id, queue_endpoint_id, weight) VALUES %s ON CONFLICT DO NOTHING",
		configQueueEndpointsTable,
		strings.Join(placeholders, ", "),
	)

	_, err := r.client.DB().ExecContext(ctx, persistence.Query{
		Name:     "crawl_job_config_repository.insertQueueEndpoints",
		QueryRaw: rawQuery,
	}, args...)
	return err
}

// replaceQueueEndpoints deletes existing join rows and inserts the new set.
func (r *crawlJobConfigRepository) replaceQueueEndpoints(ctx context.Context, configID string, assignments []models.QueueEndpointAssignment) error {
	deleteQuery := fmt.Sprintf("DELETE FROM %s WHERE crawl_job_config_id = $1", configQueueEndpointsTable)
	_, err := r.client.DB().ExecContext(ctx, persistence.Query{
		Name:     "crawl_job_config_repository.deleteQueueEndpoints",
		QueryRaw: deleteQuery,
	}, configID)
	if err != nil {
		return err
	}

	return r.insertQueueEndpoints(ctx, configID, assignments)
}

