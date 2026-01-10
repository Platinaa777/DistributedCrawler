package service

import (
	"context"
)

// Commands for Auth management

type RegisterCommand struct {
	Email    string
	Password string
}

type LoginCommand struct {
	Email    string
	Password string
}

type RefreshTokenCommand struct {
	RefreshToken string
}

type LogoutCommand struct {
	RefreshToken string
}

// Responses

type AuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int // seconds
}

// Service interface

type AuthService interface {
	Register(ctx context.Context, cmd RegisterCommand) (*AuthTokens, error)
	Login(ctx context.Context, cmd LoginCommand) (*AuthTokens, error)
	Refresh(ctx context.Context, cmd RefreshTokenCommand) (*AuthTokens, error)
	Logout(ctx context.Context, cmd LogoutCommand) error
}
