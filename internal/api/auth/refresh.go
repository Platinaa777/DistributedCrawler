package auth

import (
	"context"
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
)

func (i *AuthImplementation) Refresh(ctx context.Context, req *crawlergrpc.RefreshRequest) (*crawlergrpc.RefreshResponse, error) {
	// Build command
	command := service.RefreshTokenCommand{
		RefreshToken: req.RefreshToken,
	}

	// Execute service command
	tokens, err := i.authService.Refresh(ctx, command)
	if err != nil {
		return nil, err
	}

	// Return new tokens
	return &crawlergrpc.RefreshResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    int32(tokens.ExpiresIn),
	}, nil
}
