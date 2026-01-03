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
	nameColumn        = "name"
	statusColumn      = "status"
	createdAtColumn   = "created_at"
	completedAtColumn = "completed_at"
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
		Columns(idColumn, nameColumn, statusColumn, createdAtColumn, completedAtColumn).
		Values(dbEntity.ID, dbEntity.Name, dbEntity.Status, dbEntity.CreatedAt, dbEntity.CompletedAt).
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
	builder := sq.Select(idColumn, nameColumn, statusColumn, createdAtColumn, completedAtColumn).
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
