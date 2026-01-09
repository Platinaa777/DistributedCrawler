package snapshots

import (
	"database/sql"
	"encoding/json"
	"time"
)

// PageExtractSnapshot represents page_extract table structure
type PageExtractSnapshot struct {
	TaskID            string
	Title             sql.NullString
	MetaDescription   sql.NullString
	CanonicalURL      sql.NullString
	Metadata          json.RawMessage // JSONB
	LinkCount         int
	ImageCount        int
	ExternalLinkCount int
	WordCount         int
	ParsedAt          time.Time
	CreatedAt         time.Time
}

// PageLinkSnapshot represents page_link table structure
type PageLinkSnapshot struct {
	ID         string
	TaskID     string
	URL        string
	AnchorText sql.NullString
	IsExternal bool
	CreatedAt  time.Time
}

// PageImageSnapshot represents page_image table structure
type PageImageSnapshot struct {
	ID        string
	TaskID    string
	URL       string
	AltText   sql.NullString
	CreatedAt time.Time
}
