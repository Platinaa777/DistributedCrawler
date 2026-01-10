package models

import (
	"distributed-crawler/internal/domain/auth/valueobjects"
	"time"
)

type RefreshToken struct {
	ID                 valueobjects.RefreshTokenID
	UserID             valueobjects.UserID
	TokenHash          string
	ExpiresAt          time.Time
	RevokedAt          *time.Time
	CreatedAt          time.Time
	ReplacedByTokenID  *valueobjects.RefreshTokenID
}

func NewRefreshToken(userID valueobjects.UserID, tokenHash string, expiresAt time.Time) *RefreshToken {
	return &RefreshToken{
		ID:                valueobjects.GenerateRefreshTokenID(),
		UserID:            userID,
		TokenHash:         tokenHash,
		ExpiresAt:         expiresAt,
		RevokedAt:         nil,
		CreatedAt:         time.Now(),
		ReplacedByTokenID: nil,
	}
}

func (rt *RefreshToken) IsRevoked() bool {
	return rt.RevokedAt != nil
}

func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

func (rt *RefreshToken) IsValid() bool {
	return !rt.IsRevoked() && !rt.IsExpired()
}

func (rt *RefreshToken) Revoke() {
	now := time.Now()
	rt.RevokedAt = &now
}
