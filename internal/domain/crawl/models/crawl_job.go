package models

import (
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"time"
)

type CrawlJob struct {
	// some domain fields without db tags, only DDD logic
	ID          valueobjects.CrawlJobID
	Name        string
	Status      string
	CreatedAt   time.Time
	CompletedAt *time.Time
}
