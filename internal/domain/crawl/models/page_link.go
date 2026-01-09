package models

import (
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"time"
)

// PageLink represents an extracted link from a web page
type PageLink struct {
	ID         valueobjects.PageLinkID
	TaskID     valueobjects.CrawlTaskID
	URL        string
	AnchorText *string
	IsExternal bool
	CreatedAt  time.Time
}
