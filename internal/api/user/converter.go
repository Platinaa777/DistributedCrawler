package user

import (
	"distributed-crawler/internal/domain/auth/models"
	crawlergrpc "distributed-crawler/pkg/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoUser(user *models.User) *crawlergrpc.User {
	if user == nil {
		return nil
	}

	return &crawlergrpc.User{
		Id:        user.ID.String(),
		Email:     user.Email,
		Role:      toProtoRole(user.Role),
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}
}

func toProtoRole(role models.Role) crawlergrpc.Role {
	switch role {
	case models.RoleRead:
		return crawlergrpc.Role_ROLE_READ
	case models.RoleReadWrite:
		return crawlergrpc.Role_ROLE_READ_WRITE
	case models.RoleAdministrator:
		return crawlergrpc.Role_ROLE_ADMINISTRATOR
	default:
		return crawlergrpc.Role_ROLE_UNSPECIFIED
	}
}
