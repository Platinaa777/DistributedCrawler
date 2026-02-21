package crawljobconfig

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
)

type CrawlJobConfigRepository interface {
	Create(ctx context.Context, entity models.CrawlJobConfig) (valueobjects.ID, error)
	Get(ctx context.Context, id valueobjects.ID) (*models.CrawlJobConfig, error)
	Update(ctx context.Context, entity models.CrawlJobConfig) error
	Delete(ctx context.Context, id valueobjects.ID) error
	ListAllScheduled(ctx context.Context, limit, offset int) ([]*models.CrawlJobConfig, error)
}
