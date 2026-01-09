package page_extract

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
)

//go:generate minimock -i PageExtractRepository -o ./page_extract_repository_minimock.go -g

// PageExtractRepository manages page extract persistence
type PageExtractRepository interface {
	// Save creates or updates a page extract record (UPSERT for idempotency)
	Save(ctx context.Context, extract *models.PageExtract) error

	// GetByTaskID retrieves a page extract by task ID
	GetByTaskID(ctx context.Context, taskID valueobjects.CrawlTaskID) (*models.PageExtract, error)

	// SaveLinks saves extracted links (bulk insert with ON CONFLICT DO NOTHING)
	SaveLinks(ctx context.Context, links []*models.PageLink) error

	// SaveImages saves extracted images (bulk insert with ON CONFLICT DO NOTHING)
	SaveImages(ctx context.Context, images []*models.PageImage) error

	// GetLinksByTaskID retrieves all links for a task
	GetLinksByTaskID(ctx context.Context, taskID valueobjects.CrawlTaskID) ([]*models.PageLink, error)

	// GetImagesByTaskID retrieves all images for a task
	GetImagesByTaskID(ctx context.Context, taskID valueobjects.CrawlTaskID) ([]*models.PageImage, error)
}
