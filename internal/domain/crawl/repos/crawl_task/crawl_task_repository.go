package crawltask

import (
	"context"

	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
)

type CrawlTaskRepository interface {
	Create(ctx context.Context, entity models.CrawlTask) (valueobjects.CrawlTaskID, error)
	BulkCreate(ctx context.Context, entities []models.CrawlTask) error
	Get(ctx context.Context, id valueobjects.CrawlTaskID) (*models.CrawlTask, error)
	Update(ctx context.Context, entity models.CrawlTask) error
	ListByJob(ctx context.Context, jobID valueobjects.CrawlJobID) ([]*models.CrawlTask, error)
	ListByStatus(ctx context.Context, status models.TaskStatus, limit int) ([]*models.CrawlTask, error)

	// ListWithCursor returns tasks with cursor-based pagination and filtering
	ListWithCursor(ctx context.Context, query service.ListTasksByJobQuery) (*service.ListTasksResult, error)

	// GetAnalyticsByJob returns aggregated analytics for a job
	GetAnalyticsByJob(ctx context.Context, jobID valueobjects.CrawlJobID) (*service.TaskAnalytics, error)

	// SetTaskResult updates the result fields for a task (Part A - ParserWorker persistence)
	SetTaskResult(ctx context.Context, taskID valueobjects.CrawlTaskID, objectKey string, contentType string, sizeBytes int64) error

	// ExistsByJobIDAndHashExcluding checks if a task with the given body_hash already exists for the job,
	// excluding the specified task ID (deduplication check)
	ExistsByJobIDAndHashExcluding(ctx context.Context, jobID valueobjects.CrawlJobID, bodyHash string, excludeTaskID valueobjects.CrawlTaskID) (bool, error)
}
