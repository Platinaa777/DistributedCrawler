package snapshots

import (
	"database/sql"
	"encoding/json"
	"time"
)

// PageFetchSnapshot represents page_fetch table structure
type PageFetchSnapshot struct {
	TaskID         string
	JobID          string
	URL            string
	FinalURL       sql.NullString
	StatusCode     int
	DurationMs     int
	Headers        json.RawMessage // JSONB
	ContentType    sql.NullString
	ContentLength  sql.NullInt64
	BodyHash       string
	MinioObjectKey string
	FetchedAt      time.Time
	CreatedAt      time.Time
}
