package snapshots

import (
	"database/sql"
	"time"
)

type CrawlJobSnapshot struct {
	ID          string         `db:"id"`
	JobConfigID sql.NullString `db:"job_config_id"`
	UserID      sql.NullString `db:"user_id"`
	Name        sql.NullString `db:"name"`
	Status      string         `db:"status"`
	CreatedAt   time.Time      `db:"created_at"`
	CompletedAt sql.NullTime   `db:"completed_at"`

	// Nested job config from join
	JobConfig *CrawlJobConfigSnapshot

	// Export fields (Part B - ExportWorker)
	ExportJSONKey sql.NullString `db:"export_json_key"`
	ExportCSVKey  sql.NullString `db:"export_csv_key"`
	ExportedAt    sql.NullTime   `db:"exported_at"`
	ExportStatus  sql.NullString `db:"export_status"`
}
