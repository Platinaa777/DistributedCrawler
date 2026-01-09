package models

import (
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"time"
)

// PageImage represents an extracted image from a web page
type PageImage struct {
	ID        valueobjects.PageImageID
	TaskID    valueobjects.CrawlTaskID
	URL       string
	AltText   *string
	CreatedAt time.Time
}
