package models

import (
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"time"
)

type CrawlJob struct {
	ID          valueobjects.CrawlJobID
	Name        string
	Status      TaskStatus
	CreatedAt   time.Time
	CompletedAt *time.Time
}
