package crawltask

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
)

type CrawlTaskRepository interface {
	Create(ctx context.Context, entity models.CrawlTask) (valueobjects.CrawlTaskID, error)
	Get(ctx context.Context, id valueobjects.CrawlTaskID) (*models.CrawlTask, error)
	Update(ctx context.Context, entity models.CrawlTask) error
	ListByJob(ctx context.Context, jobID valueobjects.CrawlJobID) ([]*models.CrawlTask, error)
	ListByStatus(ctx context.Context, status models.TaskStatus, limit int) ([]*models.CrawlTask, error)
}
