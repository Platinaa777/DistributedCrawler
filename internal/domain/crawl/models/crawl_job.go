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
}
