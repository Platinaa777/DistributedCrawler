package models

import (
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"time"
)

type CrawlTask struct {
	ID         valueobjects.CrawlTaskID
	JobID      valueobjects.CrawlJobID
	Job        *CrawlJob // populated when JOIN is performed
	URL        string
	Status     TaskStatus
	EnqueuedAt time.Time
}
