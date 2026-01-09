package snapshots

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type CrawlJobConfigSnapshot struct {
	ID             string         `db:"id"`
	Name           string         `db:"name"`
	ExtractionSpec JSONB          `db:"extraction_spec"`
	Scopes         JSONB          `db:"scopes"`
	Seeds          JSONB          `db:"seeds"`
	RateLimit      JSONB          `db:"rate_limit"`
	Retries        JSONB          `db:"retries"`
	Auth           JSONB          `db:"auth"`
	Schedule       JSONB          `db:"schedule"`
}

// JSONB is a type for PostgreSQL JSONB columns
type JSONB map[string]interface{}

// Value implements driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal JSONB value")
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*j = result
	return nil
}
