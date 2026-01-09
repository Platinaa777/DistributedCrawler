package repos

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/repos/page_fetch"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/infra/persistence/postgres/converters"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

const (
	pageFetchTable = "page_fetch"

	pfTaskID         = "task_id"
	pfJobID          = "job_id"
	pfURL            = "url"
	pfFinalURL       = "final_url"
	pfStatusCode     = "status_code"
	pfDurationMs     = "duration_ms"
	pfHeaders        = "headers"
	pfContentType    = "content_type"
	pfContentLength  = "content_length"
	pfBodyHash       = "body_hash"
	pfMinioObjectKey = "minio_object_key"
	pfFetchedAt      = "fetched_at"
	pfCreatedAt      = "created_at"
)

type pageFetchRepository struct {
	client persistence.Client
}

func NewPageFetchRepository(client persistence.Client) page_fetch.PageFetchRepository {
	return &pageFetchRepository{client: client}
}

// Save creates or updates a page fetch record (UPSERT for idempotency)
func (r *pageFetchRepository) Save(ctx context.Context, fetch *models.PageFetch) error {
	snapshot, err := converters.SavePageFetchToSnapshot(fetch)
	if err != nil {
		return fmt.Errorf("failed to convert to snapshot: %w", err)
	}

	builder := sq.Insert(pageFetchTable).
		PlaceholderFormat(sq.Dollar).
		Columns(
			pfTaskID, pfJobID, pfURL, pfFinalURL, pfStatusCode, pfDurationMs,
			pfHeaders, pfContentType, pfContentLength, pfBodyHash,
			pfMinioObjectKey, pfFetchedAt, pfCreatedAt,
		).
		Values(
			snapshot.TaskID, snapshot.JobID, snapshot.URL, snapshot.FinalURL,
			snapshot.StatusCode, snapshot.DurationMs, snapshot.Headers,
			snapshot.ContentType, snapshot.ContentLength, snapshot.BodyHash,
			snapshot.MinioObjectKey, snapshot.FetchedAt, snapshot.CreatedAt,
		).
		Suffix(`
			ON CONFLICT (task_id) DO UPDATE SET
				job_id = EXCLUDED.job_id,
				url = EXCLUDED.url,
				final_url = EXCLUDED.final_url,
				status_code = EXCLUDED.status_code,
				duration_ms = EXCLUDED.duration_ms,
				headers = EXCLUDED.headers,
				content_type = EXCLUDED.content_type,
				content_length = EXCLUDED.content_length,
				body_hash = EXCLUDED.body_hash,
				minio_object_key = EXCLUDED.minio_object_key,
				fetched_at = EXCLUDED.fetched_at
		`)

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	q := persistence.Query{
		Name:     "page_fetch_repository.Save",
		QueryRaw: query,
	}

	_, err = r.client.DB().ExecContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("failed to save page fetch: %w", err)
	}

	return nil
}

// GetByTaskID retrieves a page fetch by task ID
func (r *pageFetchRepository) GetByTaskID(ctx context.Context, taskID valueobjects.CrawlTaskID) (*models.PageFetch, error) {
	builder := sq.Select(
		pfTaskID, pfJobID, pfURL, pfFinalURL, pfStatusCode, pfDurationMs,
		pfHeaders, pfContentType, pfContentLength, pfBodyHash,
		pfMinioObjectKey, pfFetchedAt, pfCreatedAt,
	).
		PlaceholderFormat(sq.Dollar).
		From(pageFetchTable).
		Where(sq.Eq{pfTaskID: taskID.String()}).
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	q := persistence.Query{
		Name:     "page_fetch_repository.GetByTaskID",
		QueryRaw: query,
	}

	var snapshot snapshots.PageFetchSnapshot
	err = r.client.DB().ScanOneContext(ctx, &snapshot, q, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get page fetch: %w", err)
	}

	return converters.RestorePageFetchFromSnapshot(&snapshot)
}

// GetByJobID retrieves all page fetches for a job
func (r *pageFetchRepository) GetByJobID(ctx context.Context, jobID valueobjects.CrawlJobID) ([]*models.PageFetch, error) {
	builder := sq.Select(
		pfTaskID, pfJobID, pfURL, pfFinalURL, pfStatusCode, pfDurationMs,
		pfHeaders, pfContentType, pfContentLength, pfBodyHash,
		pfMinioObjectKey, pfFetchedAt, pfCreatedAt,
	).
		PlaceholderFormat(sq.Dollar).
		From(pageFetchTable).
		Where(sq.Eq{pfJobID: jobID.String()}).
		OrderBy(pfFetchedAt + " DESC")

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	q := persistence.Query{
		Name:     "page_fetch_repository.GetByJobID",
		QueryRaw: query,
	}

	var snapshots []snapshots.PageFetchSnapshot
	err = r.client.DB().ScanAllContext(ctx, &snapshots, q, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get page fetches: %w", err)
	}

	fetches := make([]*models.PageFetch, 0, len(snapshots))
	for _, snapshot := range snapshots {
		fetch, err := converters.RestorePageFetchFromSnapshot(&snapshot)
		if err != nil {
			return nil, fmt.Errorf("failed to restore page fetch: %w", err)
		}
		fetches = append(fetches, fetch)
	}

	return fetches, nil
}
