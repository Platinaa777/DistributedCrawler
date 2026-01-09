package models

import (
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"time"
)

// PageExtract represents parsed HTML results and computed features
type PageExtract struct {
	TaskID             valueobjects.CrawlTaskID
	Title              *string
	MetaDescription    *string
	CanonicalURL       *string
	Metadata           map[string]any // Flexible JSONB storage
	LinkCount          int
	ImageCount         int
	ExternalLinkCount  int
	WordCount          int
	ParsedAt           time.Time
	CreatedAt          time.Time
}
