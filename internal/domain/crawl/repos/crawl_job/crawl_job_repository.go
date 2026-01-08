package crawljob

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
)

type CrawlJobRepository interface {
	Create(ctx context.Context, entity models.CrawlJob) (valueobjects.CrawlJobID, error)
	Get(ctx context.Context, id valueobjects.CrawlJobID) (*models.CrawlJob, error)
	Update(ctx context.Context, entity models.CrawlJob) error
	List(ctx context.Context, status models.TaskStatus, limit, offset int) ([]*models.CrawlJob, error)
	ListAll(ctx context.Context, limit, offset int) ([]*models.CrawlJob, error)
}