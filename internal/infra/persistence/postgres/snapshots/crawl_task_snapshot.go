package snapshots

import (
	"time"
)

type CrawlTaskSnapshot struct {
	ID         string
	JobID      string
	URL        string
	Status     string
	EnqueuedAt time.Time
}

type CrawlTaskWithJobSnapshot struct {
	CrawlTaskSnapshot
	Job *CrawlJobSnapshot
}
