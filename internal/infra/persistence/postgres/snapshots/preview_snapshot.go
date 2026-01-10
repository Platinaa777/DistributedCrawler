package snapshots

import (
	"database/sql"
	"time"
)

type PreviewSnapshot struct {
	ID          string         `db:"id"`
	SourceURL   string         `db:"source_url"`
	FinalURL    sql.NullString `db:"final_url"`
	MinioKey    string         `db:"minio_key"`
	ContentType string         `db:"content_type"`
	DownloadURL string         `db:"download_url"`
	CreatedAt   time.Time      `db:"created_at"`
	ExpiresAt   sql.NullTime   `db:"expires_at"`
}
