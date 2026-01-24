package auth

import (
	"context"
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
)

func (i *AuthImplementation) Logout(ctx context.Context, req *crawlergrpc.LogoutRequest) (*crawlergrpc.LogoutResponse, error) {
	// Build command
	command := service.LogoutCommand{
		RefreshToken: req.RefreshToken,
	}

	// Execute service command
	err := i.authService.Logout(ctx, command)
	if err != nil {
		return nil, err
	}

	// Return empty response
	return &crawlergrpc.LogoutResponse{}, nil
}
