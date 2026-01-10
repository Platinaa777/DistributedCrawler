package models

import (
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"time"
)

type CrawlTask struct {
	ID valueobjects.CrawlTaskID

	JobID valueobjects.CrawlJobID
	Job   *CrawlJob

	URL        string
	FinalURL   *string // After redirects
	Status     TaskStatus
	EnqueuedAt time.Time

	Depth          uint64
	BodyHash       string // SHA-256
	MinioObjectKey string
}

func (task *CrawlTask) MarkAsParsed(finalUrl, bodyHash, minioKey string) {
	task.BodyHash = bodyHash
	task.MinioObjectKey = minioKey
	task.FinalURL = &finalUrl

	task.Status = TaskStatusParsed
}