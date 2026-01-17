package service

import (
	"context"

	"distributed-crawler/internal/domain/auth/models"
)

type UpdateUserRoleCommand struct {
	UserID string
	Role   models.Role
}

type UserService interface {
	ListUsers(ctx context.Context) ([]*models.User, error)
	UpdateUserRole(ctx context.Context, cmd UpdateUserRoleCommand) error
}
