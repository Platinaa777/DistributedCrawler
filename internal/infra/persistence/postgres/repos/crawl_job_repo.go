package repos

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	crawljob "distributed-crawler/internal/domain/crawl/repos/crawl_job"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/infra/persistence/postgres/converters"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"
	"errors"

	sq "github.com/Masterminds/squirrel"
)

const (
	tableName = "crawl_jobs"

	idColumn          = "id"
	jobConfigIDColumn = "job_config_id"
	statusColumn      = "status"
	createdAtColumn   = "created_at"
	completedAtColumn = "completed_at"
	errorColumn       = "error"

	// Export columns
	exportJSONKeyColumn = "export_json_key"
	exportCSVKeyColumn  = "export_csv_key"
	exportedAtColumn    = "exported_at"
	exportStatusColumn  = "export_status"
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
			idColumn, jobConfigIDColumn, statusColumn, createdAtColumn, completedAtColumn, errorColumn,
			exportJSONKeyColumn, exportCSVKeyColumn, exportedAtColumn, exportStatusColumn,
		).
		Values(
			dbEntity.ID, dbEntity.JobConfigID, dbEntity.Status, dbEntity.CreatedAt, dbEntity.CompletedAt, dbEntity.Error,
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
		idColumn, jobConfigIDColumn, statusColumn, createdAtColumn, completedAtColumn, errorColumn,
		exportJSONKeyColumn, exportCSVKeyColumn, exportedAtColumn, exportStatusColumn,
	).
		PlaceholderFormat(sq.Dollar).
		From(tableName).
		Where(sq.Eq{idColumn: id.String()}).
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "crawl_job_repository.Get",
		QueryRaw: query,
	}

	var crawlJob snapshots.CrawlJobSnapshot
	err = c.client.DB().ScanOneContext(ctx, &crawlJob, q, args...)
	if err != nil {
		return nil, err
	}

	return converters.RestoreCrawlJobFromSnapshot(crawlJob)
}

func (c *crawlJobRepository) Update(ctx context.Context, entity models.CrawlJob) error {
	dbEntity := converters.SaveCrawlJobToSnapshot(entity)

	builder := sq.Update(tableName).
		PlaceholderFormat(sq.Dollar).
		Set(jobConfigIDColumn, dbEntity.JobConfigID).
		Set(statusColumn, dbEntity.Status).
		Set(completedAtColumn, dbEntity.CompletedAt).
		Set(errorColumn, dbEntity.Error).
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
		idColumn, jobConfigIDColumn, statusColumn, createdAtColumn, completedAtColumn, errorColumn,
		exportJSONKeyColumn, exportCSVKeyColumn, exportedAtColumn, exportStatusColumn,
	).
		PlaceholderFormat(sq.Dollar).
		From(tableName).
		Where(sq.Eq{statusColumn: status.String()}).
		OrderBy(createdAtColumn + " DESC").
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

	var jobSnapshots []snapshots.CrawlJobSnapshot
	err = c.client.DB().ScanAllContext(ctx, &jobSnapshots, q, args...)
	if err != nil {
		return nil, err
	}

	jobs := make([]*models.CrawlJob, 0, len(jobSnapshots))
	for _, snapshot := range jobSnapshots {
		job, err := converters.RestoreCrawlJobFromSnapshot(snapshot)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (c *crawlJobRepository) ListAll(ctx context.Context, limit, offset int) ([]*models.CrawlJob, error) {
	builder := sq.Select(
		idColumn, jobConfigIDColumn, statusColumn, createdAtColumn, completedAtColumn, errorColumn,
		exportJSONKeyColumn, exportCSVKeyColumn, exportedAtColumn, exportStatusColumn,
	).
		PlaceholderFormat(sq.Dollar).
		From(tableName).
		OrderBy(createdAtColumn + " DESC").
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

	var jobSnapshots []snapshots.CrawlJobSnapshot
	err = c.client.DB().ScanAllContext(ctx, &jobSnapshots, q, args...)
	if err != nil {
		return nil, err
	}

	jobs := make([]*models.CrawlJob, 0, len(jobSnapshots))
	for _, snapshot := range jobSnapshots {
		job, err := converters.RestoreCrawlJobFromSnapshot(snapshot)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// ListEligibleForExport finds jobs that are fully finished and not yet exported (Part B - ExportWorker)
func (c *crawlJobRepository) ListEligibleForExport(ctx context.Context, limit int) ([]*models.CrawlJob, error) {
	// A job is eligible if: completed_at IS NOT NULL AND export_status = 'NOT_STARTED'
	builder := sq.Select(
		idColumn, jobConfigIDColumn, statusColumn, createdAtColumn, completedAtColumn, errorColumn,
		exportJSONKeyColumn, exportCSVKeyColumn, exportedAtColumn, exportStatusColumn,
	).
		PlaceholderFormat(sq.Dollar).
		From(tableName).
		Where(sq.And{
			sq.NotEq{completedAtColumn: nil},
			sq.Eq{exportStatusColumn: models.ExportStatusNotStarted.String()},
		}).
		OrderBy(completedAtColumn + " ASC").
		Limit(uint64(limit))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "crawl_job_repository.ListEligibleForExport",
		QueryRaw: query,
	}

	var jobSnapshots []snapshots.CrawlJobSnapshot
	err = c.client.DB().ScanAllContext(ctx, &jobSnapshots, q, args...)
	if err != nil {
		return nil, err
	}

	jobs := make([]*models.CrawlJob, 0, len(jobSnapshots))
	for _, snapshot := range jobSnapshots {
		job, err := converters.RestoreCrawlJobFromSnapshot(snapshot)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
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

// CompleteExport updates job with export file references and marks as COMPLETED
func (c *crawlJobRepository) CompleteExport(ctx context.Context, jobID valueobjects.CrawlJobID, jsonKey, csvKey string) error {
	builder := sq.Update(tableName).
		PlaceholderFormat(sq.Dollar).
		Set(exportJSONKeyColumn, jsonKey).
		Set(exportCSVKeyColumn, csvKey).
		Set(exportedAtColumn, "NOW()").
		Set(exportStatusColumn, models.ExportStatusCompleted.String()).
		Where(sq.Eq{idColumn: jobID.String()})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := persistence.Query{
		Name:     "crawl_job_repository.CompleteExport",
		QueryRaw: query,
	}

	_, err = c.client.DB().ExecContext(ctx, q, args...)
	return err
}

// FailExport marks export as FAILED
func (c *crawlJobRepository) FailExport(ctx context.Context, jobID valueobjects.CrawlJobID, errorMsg string) error {
	// Store error message in the existing error column
	errorData := map[string]any{
		"export_error": errorMsg,
	}

	builder := sq.Update(tableName).
		PlaceholderFormat(sq.Dollar).
		Set(exportStatusColumn, models.ExportStatusFailed.String()).
		Set(errorColumn, errorData).
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
