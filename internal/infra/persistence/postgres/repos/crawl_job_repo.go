package repos

import (
	"context"
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"

	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
	crawljob "distributed-crawler/internal/domain/crawl/repos/crawl_job"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/infra/persistence/postgres/converters"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"
)

const (
	tableName = "crawl_jobs"

	idColumn          = "id"
	jobConfigIDColumn = "job_config_id"
	statusColumn      = "status"
	createdAtColumn   = "created_at"
	completedAtColumn = "completed_at"

	// Export columns
	exportJSONKeyColumn = "export_json_key"
	exportCSVKeyColumn  = "export_csv_key"
	exportedAtColumn    = "exported_at"
	exportStatusColumn  = "export_status"

	// Config table column aliases (for joins)
	aliasConfigID             = "config_id"
	aliasConfigName           = "config_name"
	aliasConfigExtractionSpec = "config_extraction_spec"
	aliasConfigScopes         = "config_scopes"
	aliasConfigSeeds          = "config_seeds"
	aliasConfigRateLimit      = "config_rate_limit"
	aliasConfigRetries        = "config_retries"
	aliasConfigAuth           = "config_auth"
	aliasConfigSchedule       = "config_schedule"
)

type crawlJobRepository struct {
	client persistence.Client
}

func NewCrawlRepository(client persistence.Client) crawljob.CrawlJobRepository {
	return &crawlJobRepository{client: client}
}

func (c *crawlJobRepository) Create(ctx context.Context, entity models.CrawlJob) (valueobjects.CrawlJobID, error) {
	dbEntity := converters.SaveCrawlJobToSnapshot(entity)

	builder := sq.Insert(tableName).
		PlaceholderFormat(sq.Dollar).
		Columns(
			idColumn, jobConfigIDColumn, statusColumn, createdAtColumn, completedAtColumn,
			exportJSONKeyColumn, exportCSVKeyColumn, exportedAtColumn, exportStatusColumn,
		).
		Values(
			dbEntity.ID, dbEntity.JobConfigID, dbEntity.Status, dbEntity.CreatedAt, dbEntity.CompletedAt,
			dbEntity.ExportJSONKey, dbEntity.ExportCSVKey, dbEntity.ExportedAt, dbEntity.ExportStatus,
		).
		Suffix("RETURNING id")

	query, args, err := builder.ToSql()
	if err != nil {
		return valueobjects.CrawlJobID{}, err
	}

	q := persistence.Query{
		Name:     "crawl_job_repository.Create",
		QueryRaw: query,
	}

	var id string
	err = c.client.DB().QueryRowContext(ctx, q, args...).Scan(&id)
	if err != nil {
		return valueobjects.CrawlJobID{}, err
	}

	return valueobjects.NewCrawlJobID(id)
}

func (c *crawlJobRepository) Get(ctx context.Context, id valueobjects.CrawlJobID) (*models.CrawlJob, error) {
	builder := sq.Select(
		"j."+idColumn, "j."+jobConfigIDColumn, "j."+statusColumn, "j."+createdAtColumn, "j."+completedAtColumn,
		"j."+exportJSONKeyColumn, "j."+exportCSVKeyColumn, "j."+exportedAtColumn, "j."+exportStatusColumn,
		"c.id as "+aliasConfigID, "c.name as "+aliasConfigName,
		"c.extraction_spec as "+aliasConfigExtractionSpec, "c.scopes as "+aliasConfigScopes,
		"c.seeds as "+aliasConfigSeeds, "c.rate_limit as "+aliasConfigRateLimit,
		"c.retries as "+aliasConfigRetries, "c.auth as "+aliasConfigAuth,
		"c.schedule as "+aliasConfigSchedule,
	).
		PlaceholderFormat(sq.Dollar).
		From(tableName + " j").
		LeftJoin(configTableName + " c ON j." + jobConfigIDColumn + " = c.id").
		Where(sq.Eq{"j." + idColumn: id.String()}).
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "crawl_job_repository.Get",
		QueryRaw: query,
	}

	row := c.client.DB().QueryRowContext(ctx, q, args...)
	crawlJob, err := scanCrawlJobWithConfig(row)
	if err != nil {
		return nil, err
	}

	return converters.RestoreCrawlJobFromSnapshot(*crawlJob)
}

func (c *crawlJobRepository) Update(ctx context.Context, entity models.CrawlJob) error {
	dbEntity := converters.SaveCrawlJobToSnapshot(entity)

	builder := sq.Update(tableName).
		PlaceholderFormat(sq.Dollar).
		Set(jobConfigIDColumn, dbEntity.JobConfigID).
		Set(statusColumn, dbEntity.Status).
		Set(completedAtColumn, dbEntity.CompletedAt).
		Set(exportJSONKeyColumn, dbEntity.ExportJSONKey).
		Set(exportCSVKeyColumn, dbEntity.ExportCSVKey).
		Set(exportedAtColumn, dbEntity.ExportedAt).
		Set(exportStatusColumn, dbEntity.ExportStatus).
		Where(sq.Eq{idColumn: dbEntity.ID})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := persistence.Query{
		Name:     "crawl_job_repository.Update",
		QueryRaw: query,
	}

	_, err = c.client.DB().ExecContext(ctx, q, args...)
	return err
}

func (c *crawlJobRepository) List(ctx context.Context, status models.TaskStatus, limit, offset int) ([]*models.CrawlJob, error) {
	builder := sq.Select(
		"j."+idColumn, "j."+jobConfigIDColumn, "j."+statusColumn, "j."+createdAtColumn, "j."+completedAtColumn,
		"j."+exportJSONKeyColumn, "j."+exportCSVKeyColumn, "j."+exportedAtColumn, "j."+exportStatusColumn,
		"c.id as "+aliasConfigID, "c.name as "+aliasConfigName,
		"c.extraction_spec as "+aliasConfigExtractionSpec, "c.scopes as "+aliasConfigScopes,
		"c.seeds as "+aliasConfigSeeds, "c.rate_limit as "+aliasConfigRateLimit,
		"c.retries as "+aliasConfigRetries, "c.auth as "+aliasConfigAuth,
		"c.schedule as "+aliasConfigSchedule,
	).
		PlaceholderFormat(sq.Dollar).
		From(tableName + " j").
		LeftJoin(configTableName + " c ON j." + jobConfigIDColumn + " = c.id").
		Where(sq.Eq{"j." + statusColumn: status.String()}).
		OrderBy("j." + createdAtColumn + " DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "crawl_job_repository.List",
		QueryRaw: query,
	}

	rows, err := c.client.DB().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]*models.CrawlJob, 0, limit)
	for rows.Next() {
		snapshot, err := scanCrawlJobWithConfig(rows)
		if err != nil {
			return nil, err
		}
		job, err := converters.RestoreCrawlJobFromSnapshot(*snapshot)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

func (c *crawlJobRepository) ListAll(ctx context.Context, limit, offset int) ([]*models.CrawlJob, error) {
	builder := sq.Select(
		"j."+idColumn, "j."+jobConfigIDColumn, "j."+statusColumn, "j."+createdAtColumn, "j."+completedAtColumn,
		"j."+exportJSONKeyColumn, "j."+exportCSVKeyColumn, "j."+exportedAtColumn, "j."+exportStatusColumn,
		"c.id as "+aliasConfigID, "c.name as "+aliasConfigName,
		"c.extraction_spec as "+aliasConfigExtractionSpec, "c.scopes as "+aliasConfigScopes,
		"c.seeds as "+aliasConfigSeeds, "c.rate_limit as "+aliasConfigRateLimit,
		"c.retries as "+aliasConfigRetries, "c.auth as "+aliasConfigAuth,
		"c.schedule as "+aliasConfigSchedule,
	).
		PlaceholderFormat(sq.Dollar).
		From(tableName + " j").
		LeftJoin(configTableName + " c ON j." + jobConfigIDColumn + " = c.id").
		OrderBy("j." + createdAtColumn + " DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "crawl_job_repository.ListAll",
		QueryRaw: query,
	}

	rows, err := c.client.DB().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]*models.CrawlJob, 0, limit)
	for rows.Next() {
		snapshot, err := scanCrawlJobWithConfig(rows)
		if err != nil {
			return nil, err
		}
		job, err := converters.RestoreCrawlJobFromSnapshot(*snapshot)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

// ListWithCursor returns jobs with cursor-based pagination and filtering
func (c *crawlJobRepository) ListWithCursor(ctx context.Context, query service.ListCrawlJobsQuery) (*service.ListCrawlJobsResult, error) {
	// Fetch one extra row to determine if there are more results
	fetchLimit := query.Limit + 1

	builder := sq.Select(
		"j."+idColumn, "j."+jobConfigIDColumn, "j."+statusColumn, "j."+createdAtColumn, "j."+completedAtColumn,
		"j."+exportJSONKeyColumn, "j."+exportCSVKeyColumn, "j."+exportedAtColumn, "j."+exportStatusColumn,
		"c.id as "+aliasConfigID, "c.name as "+aliasConfigName,
		"c.extraction_spec as "+aliasConfigExtractionSpec, "c.scopes as "+aliasConfigScopes,
		"c.seeds as "+aliasConfigSeeds, "c.rate_limit as "+aliasConfigRateLimit,
		"c.retries as "+aliasConfigRetries, "c.auth as "+aliasConfigAuth,
		"c.schedule as "+aliasConfigSchedule,
	).
		PlaceholderFormat(sq.Dollar).
		From(tableName + " j").
		LeftJoin(configTableName + " c ON j." + jobConfigIDColumn + " = c.id")

	// Build WHERE conditions
	conditions := sq.And{}

	// Filter by status
	if query.Filter.Status != nil && *query.Filter.Status != "" {
		conditions = append(conditions, sq.Eq{"j." + statusColumn: *query.Filter.Status})
	}

	// Filter by name (partial match, case-insensitive)
	if query.Filter.Name != nil && *query.Filter.Name != "" {
		conditions = append(conditions, sq.ILike{"c.name": "%" + *query.Filter.Name + "%"})
	}

	// Filter by created_at range
	if query.Filter.CreatedFrom != nil {
		conditions = append(conditions, sq.GtOrEq{"j." + createdAtColumn: *query.Filter.CreatedFrom})
	}
	if query.Filter.CreatedTo != nil {
		conditions = append(conditions, sq.LtOrEq{"j." + createdAtColumn: *query.Filter.CreatedTo})
	}

	// Apply cursor condition (seek method)
	// For DESC ordering: WHERE (created_at, id) < (cursor.created_at, cursor.id)
	if query.Cursor != nil {
		cursorCondition := sq.Expr(
			"(j."+createdAtColumn+", j."+idColumn+") < (?, ?)",
			query.Cursor.CreatedAt,
			query.Cursor.ID,
		)
		conditions = append(conditions, cursorCondition)
	}

	if len(conditions) > 0 {
		builder = builder.Where(conditions)
	}

	// Order by created_at DESC, id DESC for stable ordering
	builder = builder.OrderBy("j."+createdAtColumn+" DESC", "j."+idColumn+" DESC")
	builder = builder.Limit(uint64(fetchLimit))

	sqlQuery, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "crawl_job_repository.ListWithCursor",
		QueryRaw: sqlQuery,
	}

	rows, err := c.client.DB().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]*models.CrawlJob, 0, query.Limit)
	for rows.Next() {
		snapshot, err := scanCrawlJobWithConfig(rows)
		if err != nil {
			return nil, err
		}
		job, err := converters.RestoreCrawlJobFromSnapshot(*snapshot)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Determine if there are more results
	hasMore := len(jobs) > query.Limit
	if hasMore {
		jobs = jobs[:query.Limit] // Trim extra row
	}

	// Build next cursor from last item
	var nextCursor *service.ListCrawlJobsCursor
	if hasMore && len(jobs) > 0 {
		lastJob := jobs[len(jobs)-1]
		nextCursor = &service.ListCrawlJobsCursor{
			CreatedAt: lastJob.CreatedAt,
			ID:        lastJob.ID.String(),
		}
	}

	return &service.ListCrawlJobsResult{
		Jobs:       jobs,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (c *crawlJobRepository) GetLatestByConfigID(ctx context.Context, configID valueobjects.ID) (*models.CrawlJob, error) {
	builder := sq.Select(
		"j."+idColumn, "j."+jobConfigIDColumn, "j."+statusColumn, "j."+createdAtColumn, "j."+completedAtColumn,
		"j."+exportJSONKeyColumn, "j."+exportCSVKeyColumn, "j."+exportedAtColumn, "j."+exportStatusColumn,
		"c.id as "+aliasConfigID, "c.name as "+aliasConfigName,
		"c.extraction_spec as "+aliasConfigExtractionSpec, "c.scopes as "+aliasConfigScopes,
		"c.seeds as "+aliasConfigSeeds, "c.rate_limit as "+aliasConfigRateLimit,
		"c.retries as "+aliasConfigRetries, "c.auth as "+aliasConfigAuth,
		"c.schedule as "+aliasConfigSchedule,
	).
		PlaceholderFormat(sq.Dollar).
		From(tableName + " j").
		LeftJoin(configTableName + " c ON j." + jobConfigIDColumn + " = c.id").
		Where(sq.Eq{"j." + jobConfigIDColumn: configID.String()}).
		OrderBy("j." + createdAtColumn + " DESC").
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "crawl_job_repository.GetLatestByConfigID",
		QueryRaw: query,
	}

	row := c.client.DB().QueryRowContext(ctx, q, args...)
	crawlJob, err := scanCrawlJobWithConfig(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return converters.RestoreCrawlJobFromSnapshot(*crawlJob)
}

// ListEligibleForExport finds jobs that are fully finished and not yet exported (Part B - ExportWorker)
func (c *crawlJobRepository) ListEligibleForExport(ctx context.Context, limit int) ([]*models.CrawlJob, error) {
	// A job is eligible if: all tasks are Parsed or Completed AND export_status = 'NOT_STARTED'
	builder := sq.Select(
		"j."+idColumn, "j."+jobConfigIDColumn, "j."+statusColumn, "j."+createdAtColumn, "j."+completedAtColumn,
		"j."+exportJSONKeyColumn, "j."+exportCSVKeyColumn, "j."+exportedAtColumn, "j."+exportStatusColumn,
		"c.id as "+aliasConfigID, "c.name as "+aliasConfigName,
		"c.extraction_spec as "+aliasConfigExtractionSpec, "c.scopes as "+aliasConfigScopes,
		"c.seeds as "+aliasConfigSeeds, "c.rate_limit as "+aliasConfigRateLimit,
		"c.retries as "+aliasConfigRetries, "c.auth as "+aliasConfigAuth,
		"c.schedule as "+aliasConfigSchedule,
	).
		PlaceholderFormat(sq.Dollar).
		From(tableName + " j").
		LeftJoin(configTableName + " c ON j." + jobConfigIDColumn + " = c.id").
		Where(sq.And{
			sq.NotEq{"j." + exportStatusColumn: models.ExportStatusCompleted.String()},
			sq.Expr(
				"NOT EXISTS (SELECT 1 FROM "+taskTableName+" t WHERE t."+taskJobIDColumn+" = j."+idColumn+" AND t."+taskStatusColumn+" NOT IN (?, ?))",
				models.TaskStatusParsed.String(),
				models.TaskStatusCompleted.String(),
			),
		}).
		OrderBy("j." + completedAtColumn + " ASC").
		Limit(uint64(limit)).
		Suffix("FOR UPDATE OF j SKIP LOCKED")

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "crawl_job_repository.ListEligibleForExport",
		QueryRaw: query,
	}

	rows, err := c.client.DB().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]*models.CrawlJob, 0, limit)
	for rows.Next() {
		snapshot, err := scanCrawlJobWithConfig(rows)
		if err != nil {
			return nil, err
		}
		job, err := converters.RestoreCrawlJobFromSnapshot(*snapshot)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

// TryStartExport atomically transitions export_status from NOT_STARTED to IN_PROGRESS
func (c *crawlJobRepository) TryStartExport(ctx context.Context, jobID valueobjects.CrawlJobID) (bool, error) {
	// Use UPDATE with WHERE condition to ensure atomicity (compare-and-swap)
	builder := sq.Update(tableName).
		PlaceholderFormat(sq.Dollar).
		Set(exportStatusColumn, models.ExportStatusInProgress.String()).
		Where(sq.And{
			sq.Eq{idColumn: jobID.String()},
			sq.Eq{exportStatusColumn: models.ExportStatusNotStarted.String()},
		})

	query, args, err := builder.ToSql()
	if err != nil {
		return false, err
	}

	q := persistence.Query{
		Name:     "crawl_job_repository.TryStartExport",
		QueryRaw: query,
	}

	result, err := c.client.DB().ExecContext(ctx, q, args...)
	if err != nil {
		return false, err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected != 1 {
		return false, errors.New("more than 1 row affected")
	}

	// If 1 row was affected, we successfully transitioned to IN_PROGRESS
	return rowsAffected == 1, nil
}

// FailExport marks export as FAILED
func (c *crawlJobRepository) FailExport(ctx context.Context, jobID valueobjects.CrawlJobID, errorMsg string) error {
	builder := sq.Update(tableName).
		PlaceholderFormat(sq.Dollar).
		Set(exportStatusColumn, models.ExportStatusFailed.String()).
		Where(sq.Eq{idColumn: jobID.String()})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := persistence.Query{
		Name:     "crawl_job_repository.FailExport",
		QueryRaw: query,
	}

	_, err = c.client.DB().ExecContext(ctx, q, args...)
	return err
}

// scanCrawlJobWithConfig scans a row containing joined crawl_job and crawl_job_config data
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanCrawlJobWithConfig(row scanner) (*snapshots.CrawlJobSnapshot, error) {
	var job snapshots.CrawlJobSnapshot
	var config snapshots.CrawlJobConfigSnapshot

	err := row.Scan(
		// Job fields
		&job.ID,
		&job.JobConfigID,
		&job.Status,
		&job.CreatedAt,
		&job.CompletedAt,
		&job.ExportJSONKey,
		&job.ExportCSVKey,
		&job.ExportedAt,
		&job.ExportStatus,
		// Config fields (may be NULL if LEFT JOIN returns no match)
		&config.ID,
		&config.Name,
		&config.ExtractionSpec,
		&config.Scopes,
		&config.Seeds,
		&config.RateLimit,
		&config.Retries,
		&config.Auth,
		&config.Schedule,
	)
	if err != nil {
		return nil, err
	}

	// Only attach config if it exists (ID is not empty)
	if config.ID != "" {
		job.JobConfig = &config
	}

	return &job, nil
}
