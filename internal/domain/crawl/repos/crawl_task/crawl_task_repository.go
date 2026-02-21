package crawltask

import (
	"context"

	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
)

type CrawlTaskRepository interface {
	Create(ctx context.Context, entity models.CrawlTask) (valueobjects.CrawlTaskID, error)
	// BulkCreate inserts multiple tasks atomically, ignoring duplicates (same job_id + url).
	// Returns the IDs of rows that were actually inserted (conflicts are silently skipped).
	BulkCreate(ctx context.Context, entities []models.CrawlTask) ([]valueobjects.CrawlTaskID, error)
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

	// ExistsByJobIDAndURL checks if a task with the given URL already exists for the job (URL deduplication)
	ExistsByJobIDAndURL(ctx context.Context, jobID valueobjects.CrawlJobID, url string) (bool, error)
}
