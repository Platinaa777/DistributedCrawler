package repos

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	crawljob "distributed-crawler/internal/domain/crawl/repos/crawl_job"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/infra/persistence/postgres/converters"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"

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
		Columns(idColumn, jobConfigIDColumn, statusColumn, createdAtColumn, completedAtColumn, errorColumn).
		Values(dbEntity.ID, dbEntity.JobConfigID, dbEntity.Status, dbEntity.CreatedAt, dbEntity.CompletedAt, dbEntity.Error).
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
	builder := sq.Select(idColumn, jobConfigIDColumn, statusColumn, createdAtColumn, completedAtColumn, errorColumn).
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
	builder := sq.Select(idColumn, jobConfigIDColumn, statusColumn, createdAtColumn, completedAtColumn, errorColumn).
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
	builder := sq.Select(idColumn, jobConfigIDColumn, statusColumn, createdAtColumn, completedAtColumn, errorColumn).
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
