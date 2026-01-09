package models

import (
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"time"
)

// PageFetch represents fetch metadata and MinIO storage reference
type PageFetch struct {
	TaskID          valueobjects.CrawlTaskID
	JobID           valueobjects.CrawlJobID
	URL             string
	FinalURL        *string // After redirects
	StatusCode      int
	DurationMs      int
	Headers         map[string]string
	ContentType     *string
	ContentLength   *int64
	BodyHash        string // SHA-256
	MinioObjectKey  string
	FetchedAt       time.Time
	CreatedAt       time.Time
}
