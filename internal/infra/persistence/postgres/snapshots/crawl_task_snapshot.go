package snapshots

import (
	"database/sql"
	"time"
)

type CrawlTaskSnapshot struct {
	ID             string         `db:"id"`
	JobID          string         `db:"job_id"`
	URL            string         `db:"url"`
	FinalURL       sql.NullString `db:"final_url"`
	Status         string         `db:"status"`
	EnqueuedAt     time.Time      `db:"enqueued_at"`
	Depth          uint64         `db:"depth"`
	BodyHash       string         `db:"body_hash"`
	MinioObjectKey string         `db:"minio_object_key"`
}

type CrawlTaskWithJobSnapshot struct {
	CrawlTaskSnapshot
	Job *CrawlJobSnapshot
}
