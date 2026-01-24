package converters

import (
	"database/sql"
	"distributed-crawler/internal/domain/auth/models"
	"distributed-crawler/internal/domain/auth/valueobjects"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"
	"time"
)

func RefreshTokenSnapshotToModel(snapshot snapshots.RefreshTokenSnapshot) (*models.RefreshToken, error) {
	tokenID, err := valueobjects.NewRefreshTokenID(snapshot.ID)
	if err != nil {
		return nil, err
	}

	userID, err := valueobjects.NewUserID(snapshot.UserID)
	if err != nil {
		return nil, err
	}

	var revokedAt *time.Time
	if snapshot.RevokedAt.Valid {
		revokedAt = &snapshot.RevokedAt.Time
	}

	var replacedByTokenID *valueobjects.RefreshTokenID
	if snapshot.ReplacedByTokenID.Valid {
		rid, err := valueobjects.NewRefreshTokenID(snapshot.ReplacedByTokenID.String)
		if err != nil {
			return nil, err
		}
		replacedByTokenID = &rid
	}

	return &models.RefreshToken{
		ID:                tokenID,
		UserID:            userID,
		TokenHash:         snapshot.TokenHash,
		ExpiresAt:         snapshot.ExpiresAt,
		RevokedAt:         revokedAt,
		CreatedAt:         snapshot.CreatedAt,
		ReplacedByTokenID: replacedByTokenID,
	}, nil
}

func RefreshTokenModelToSnapshot(token *models.RefreshToken) snapshots.RefreshTokenSnapshot {
	var revokedAt sql.NullTime
	if token.RevokedAt != nil {
		revokedAt = sql.NullTime{
			Time:  *token.RevokedAt,
			Valid: true,
		}
	}

	var replacedByTokenID sql.NullString
	if token.ReplacedByTokenID != nil {
		replacedByTokenID = sql.NullString{
			String: token.ReplacedByTokenID.String(),
			Valid:  true,
		}
	}

	return snapshots.RefreshTokenSnapshot{
		ID:                token.ID.String(),
		UserID:            token.UserID.String(),
		TokenHash:         token.TokenHash,
		ExpiresAt:         token.ExpiresAt,
		RevokedAt:         revokedAt,
		CreatedAt:         token.CreatedAt,
		ReplacedByTokenID: replacedByTokenID,
	}
}
