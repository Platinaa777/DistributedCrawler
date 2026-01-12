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
	BodyHash       sql.NullString `db:"body_hash"`
	MinioObjectKey string         `db:"minio_object_key"`

	// Result persistence fields
	ResultObjectKey   sql.NullString `db:"result_object_key"`
	ResultContentType sql.NullString `db:"result_content_type"`
	ResultSizeBytes   sql.NullInt64  `db:"result_size_bytes"`
	ResultCreatedAt   sql.NullTime   `db:"result_created_at"`
}

type CrawlTaskWithJobSnapshot struct {
	CrawlTaskSnapshot
	Job *CrawlJobSnapshot
}
