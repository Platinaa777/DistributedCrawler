package page_fetch

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
)

//go:generate minimock -i PageFetchRepository -o ./page_fetch_repository_minimock.go -g

// PageFetchRepository manages page fetch metadata persistence
type PageFetchRepository interface {
	// Save creates or updates a page fetch record (UPSERT for idempotency)
	Save(ctx context.Context, fetch *models.PageFetch) error

	// GetByTaskID retrieves a page fetch by task ID
	GetByTaskID(ctx context.Context, taskID valueobjects.CrawlTaskID) (*models.PageFetch, error)

	// GetByJobID retrieves all page fetches for a job
	GetByJobID(ctx context.Context, jobID valueobjects.CrawlJobID) ([]*models.PageFetch, error)
}
