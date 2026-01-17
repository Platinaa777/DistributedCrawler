package user

import (
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
)

type UserImplementation struct {
	crawlergrpc.UnimplementedUserServiceServer
	userService service.UserService
}

func NewImplementation(userService service.UserService) *UserImplementation {
	return &UserImplementation{
		userService: userService,
	}
}
