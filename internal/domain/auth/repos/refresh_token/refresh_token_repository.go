package refreshtoken

import (
	"context"
	"distributed-crawler/internal/domain/auth/models"
	"distributed-crawler/internal/domain/auth/valueobjects"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, entity *models.RefreshToken) (valueobjects.RefreshTokenID, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	GetByID(ctx context.Context, id valueobjects.RefreshTokenID) (*models.RefreshToken, error)
	Update(ctx context.Context, entity *models.RefreshToken) error
	RevokeByTokenHash(ctx context.Context, tokenHash string) error
	RevokeAllByUserID(ctx context.Context, userID valueobjects.UserID) error
}
