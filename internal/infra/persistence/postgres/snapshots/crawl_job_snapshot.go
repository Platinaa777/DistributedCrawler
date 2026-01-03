package snapshots

import (
	"database/sql"
	"time"
)

type CrawlJobSnapshot struct {
	ID          string
	Name        string
	Status      string
	CreatedAt   time.Time
	CompletedAt sql.NullTime
}
