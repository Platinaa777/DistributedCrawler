package models

import (
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"time"
)

type PageSnapshot struct {
	ID          valueobjects.PageSnapshotID
	TaskID      valueobjects.CrawlTaskID
	URL         string
	HTTPStatus  int
	ContentType *string
	StorageKey  string
	FetchedAt   time.Time
}
