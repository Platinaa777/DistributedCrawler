package models

import (
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"time"
)

type ExtractedRecord struct {
	ID        valueobjects.ExtractedRecordID
	TaskID    valueobjects.CrawlTaskID
	SourceURL string
	Data      map[string]any
	ParsedAt  time.Time
}
