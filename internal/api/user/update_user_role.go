package user

import (
	"context"

	"distributed-crawler/internal/application/service"
	userservice "distributed-crawler/internal/application/service/user"
	"distributed-crawler/internal/domain/auth/models"
	crawlergrpc "distributed-crawler/pkg/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i *UserImplementation) UpdateUserRole(ctx context.Context, req *crawlergrpc.UpdateUserRoleRequest) (*crawlergrpc.UpdateUserRoleResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	role, err := fromProtoRole(req.GetRole())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid role")
	}

	err = i.userService.UpdateUserRole(ctx, service.UpdateUserRoleCommand{
		UserID: req.GetId(),
		Role:   role,
	})
	if err != nil {
		switch err {
		case userservice.ErrInvalidRole, userservice.ErrInvalidUserID:
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case userservice.ErrUserNotFound:
			return nil, status.Error(codes.NotFound, err.Error())
		case userservice.ErrRoleNotAllowed:
			return nil, status.Error(codes.PermissionDenied, err.Error())
		case userservice.ErrRoleUnchanged:
			return &crawlergrpc.UpdateUserRoleResponse{Updated: false}, nil
		default:
			return nil, status.Error(codes.Internal, "failed to update user role")
		}
	}

	return &crawlergrpc.UpdateUserRoleResponse{Updated: true}, nil
}

func fromProtoRole(role crawlergrpc.Role) (models.Role, error) {
	switch role {
	case crawlergrpc.Role_ROLE_READ:
		return models.RoleRead, nil
	case crawlergrpc.Role_ROLE_READ_WRITE:
		return models.RoleReadWrite, nil
	case crawlergrpc.Role_ROLE_ADMINISTRATOR:
		return models.RoleAdministrator, nil
	default:
		return models.ParseRole("")
	}
}
