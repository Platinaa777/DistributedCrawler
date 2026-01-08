package models

import (
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"time"
)

type CrawlTask struct {
	ID         valueobjects.CrawlTaskID
	JobID      valueobjects.CrawlJobID
	URL        string
	Status     TaskStatus
	EnqueuedAt time.Time
}
