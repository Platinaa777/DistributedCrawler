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

	// Export-related methods (Part B - ExportWorker)
	// ListEligibleForExport finds jobs that are fully finished and not yet exported
	ListEligibleForExport(ctx context.Context, limit int) ([]*models.CrawlJob, error)

	// TryStartExport atomically transitions export_status from NOT_STARTED to IN_PROGRESS
	// Returns true if successful, false if already in progress or completed
	TryStartExport(ctx context.Context, jobID valueobjects.CrawlJobID) (bool, error)

	// CompleteExport updates job with export file references and marks as COMPLETED
	CompleteExport(ctx context.Context, jobID valueobjects.CrawlJobID, jsonKey, csvKey string) error

	// FailExport marks export as FAILED
	FailExport(ctx context.Context, jobID valueobjects.CrawlJobID, errorMsg string) error
}
