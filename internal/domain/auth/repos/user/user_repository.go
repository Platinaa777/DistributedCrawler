package user

import (
	"context"
	"distributed-crawler/internal/domain/auth/models"
	"distributed-crawler/internal/domain/auth/valueobjects"
)

type UserRepository interface {
	Create(ctx context.Context, entity *models.User) (valueobjects.UserID, error)
	GetByID(ctx context.Context, id valueobjects.UserID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, entity *models.User) error
	List(ctx context.Context) ([]*models.User, error)
	UpdateRole(ctx context.Context, id valueobjects.UserID, role models.Role) error
}
