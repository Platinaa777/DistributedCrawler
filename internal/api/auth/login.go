package auth

import (
	"context"
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
)

func (i *AuthImplementation) Login(ctx context.Context, req *crawlergrpc.LoginRequest) (*crawlergrpc.LoginResponse, error) {
	// Build command
	command := service.LoginCommand{
		Email:    req.Email,
		Password: req.Password,
	}

	// Execute service command
	tokens, err := i.authService.Login(ctx, command)
	if err != nil {
		return nil, err
	}

	// Return tokens
	return &crawlergrpc.LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    int32(tokens.ExpiresIn),
	}, nil
}
