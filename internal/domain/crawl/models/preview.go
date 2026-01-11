package models

import (
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"time"
)

// Preview represents a sanitized HTML preview for UI inspection
type Preview struct {
	ID          valueobjects.PreviewID
	SourceURL   string
	FinalURL    *string
	MinioKey    string
	ContentType string
	DownloadURL string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
}
