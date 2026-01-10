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

	Error map[string]any

	// Export fields (Part B - ExportWorker)
	ExportJSONKey *string        // S3 object key for JSON export
	ExportCSVKey  *string        // S3 object key for CSV export
	ExportedAt    *time.Time     // When export was completed
	ExportStatus  ExportStatus   // Export status (NOT_STARTED, IN_PROGRESS, COMPLETED, FAILED)
}
