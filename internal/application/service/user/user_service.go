package user

import (
	"context"
	"errors"

	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/auth/models"
	userrepo "distributed-crawler/internal/domain/auth/repos/user"
	"distributed-crawler/internal/domain/auth/valueobjects"
	"distributed-crawler/internal/infra/persistence"
)

var (
	ErrInvalidRole    = errors.New("invalid role")
	ErrRoleNotAllowed = errors.New("role is not allowed")
	ErrInvalidUserID  = errors.New("invalid user id")
	ErrUserNotFound   = errors.New("user not found")
	ErrRoleUnchanged  = errors.New("role is already assigned")
)

type userService struct {
	userRepo  userrepo.UserRepository
	txManager persistence.TxManager
}

func NewUserService(userRepo userrepo.UserRepository, txManager persistence.TxManager) service.UserService {
	return &userService{
		userRepo:  userRepo,
		txManager: txManager,
	}
}

func (s *userService) ListUsers(ctx context.Context) ([]*models.User, error) {
	return s.userRepo.List(ctx)
}

func (s *userService) UpdateUserRole(ctx context.Context, cmd service.UpdateUserRoleCommand) error {
	if !cmd.Role.IsValid() {
		return ErrInvalidRole
	}
	if cmd.Role == models.RoleAdministrator {
		return ErrRoleNotAllowed
	}

	userID, err := valueobjects.NewUserID(cmd.UserID)
	if err != nil {
		return ErrInvalidUserID
	}

	return s.txManager.ReadCommitted(ctx, func(ctx context.Context) error {
		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil {
			return err
		}
		if user == nil {
			return ErrUserNotFound
		}
		if user.Role == cmd.Role {
			return ErrRoleUnchanged
		}

		return s.userRepo.UpdateRole(ctx, userID, cmd.Role)
	})
}
