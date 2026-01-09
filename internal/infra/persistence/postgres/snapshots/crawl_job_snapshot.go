package snapshots

import (
	"database/sql"
	"time"
)

type CrawlJobSnapshot struct {
	ID          string         `db:"id"`
	JobConfigID sql.NullString `db:"job_config_id"`
	Status      string         `db:"status"`
	CreatedAt   time.Time      `db:"created_at"`
	CompletedAt sql.NullTime   `db:"completed_at"`
	Error       JSONB          `db:"error"`
}
