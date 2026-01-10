package auth

import (
	"context"
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
)

func (i *AuthImplementation) Register(ctx context.Context, req *crawlergrpc.RegisterRequest) (*crawlergrpc.RegisterResponse, error) {
	// Build command
	command := service.RegisterCommand{
		Email:    req.Email,
		Password: req.Password,
	}

	// Execute service command
	tokens, err := i.authService.Register(ctx, command)
	if err != nil {
		return nil, err
	}

	// Return tokens
	return &crawlergrpc.RegisterResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    int32(tokens.ExpiresIn),
	}, nil
}
