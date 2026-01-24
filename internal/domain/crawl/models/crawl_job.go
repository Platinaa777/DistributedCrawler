package models

import (
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"time"
)

type CrawlJob struct {
	ID valueobjects.CrawlJobID

	JobConfigID valueobjects.ID
	JobConfig   *CrawlJobConfig

	Status TaskStatus

	CreatedAt   time.Time
	CompletedAt *time.Time

	// Export fields (Part B - ExportWorker)
	ExportJSONKey *string      // S3 object key for JSON export
	ExportCSVKey  *string      // S3 object key for CSV export
	ExportedAt    *time.Time   // When export was completed
	ExportStatus  ExportStatus // Export status (NOT_STARTED, IN_PROGRESS, COMPLETED, FAILED)
}

func (job *CrawlJob) MarkAsExported(jsonKey, csvKey string, exportedAt time.Time) {
	job.ExportJSONKey = &jsonKey
	job.ExportCSVKey = &csvKey
	job.ExportedAt = &exportedAt
	job.CompletedAt = &exportedAt
	job.ExportStatus = ExportStatusCompleted
	job.Status = TaskStatusCompleted
}
