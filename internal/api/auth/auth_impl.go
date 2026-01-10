package auth

import (
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
)

type AuthImplementation struct {
	crawlergrpc.UnimplementedAuthServiceServer
	authService service.AuthService
}

func NewImplementation(authService service.AuthService) *AuthImplementation {
	return &AuthImplementation{
		authService: authService,
	}
}
