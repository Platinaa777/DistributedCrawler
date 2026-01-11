package snapshots

import (
	"database/sql"
	"time"
)

type RefreshTokenSnapshot struct {
	ID                string         `db:"id"`
	UserID            string         `db:"user_id"`
	TokenHash         string         `db:"token_hash"`
	ExpiresAt         time.Time      `db:"expires_at"`
	RevokedAt         sql.NullTime   `db:"revoked_at"`
	CreatedAt         time.Time      `db:"created_at"`
	ReplacedByTokenID sql.NullString `db:"replaced_by_token_id"`
}
