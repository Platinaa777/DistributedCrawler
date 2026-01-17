package auth

import (
	"context"
	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/auth"
	"distributed-crawler/internal/domain/auth/models"
	refreshtokenrepo "distributed-crawler/internal/domain/auth/repos/refresh_token"
	userrepo "distributed-crawler/internal/domain/auth/repos/user"
	"distributed-crawler/internal/infra/persistence"
	"errors"
	"fmt"
	"net/mail"
	"time"
)

var (
	ErrUserExists          = errors.New("user already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrPasswordTooShort    = errors.New("password must be at least 8 characters")
	ErrInvalidEmail        = errors.New("invalid email format")
)

type authService struct {
	userRepo         userrepo.UserRepository
	refreshTokenRepo refreshtokenrepo.RefreshTokenRepository
	txManager        persistence.TxManager
	jwtService       *auth.JWTService
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
}

func NewAuthService(
	userRepo userrepo.UserRepository,
	refreshTokenRepo refreshtokenrepo.RefreshTokenRepository,
	txManager persistence.TxManager,
	jwtService *auth.JWTService,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
) service.AuthService {
	return &authService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		txManager:        txManager,
		jwtService:       jwtService,
		accessTokenTTL:   accessTokenTTL,
		refreshTokenTTL:  refreshTokenTTL,
	}
}

func (s *authService) Register(ctx context.Context, cmd service.RegisterCommand) (*service.AuthTokens, error) {
	// Validate email
	if _, err := mail.ParseAddress(cmd.Email); err != nil {
		return nil, ErrInvalidEmail
	}

	// Validate password
	if len(cmd.Password) < 8 {
		return nil, ErrPasswordTooShort
	}

	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, cmd.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, ErrUserExists
	}

	// Hash password
	passwordHash, err := auth.HashPassword(cmd.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := models.NewUser(cmd.Email, passwordHash)

	var tokens *service.AuthTokens
	err = s.txManager.ReadCommitted(ctx, func(ctx context.Context) error {
		// Save user
		_, err := s.userRepo.Create(ctx, user)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		// Generate tokens
		tokens, err = s.generateTokens(ctx, user)
		if err != nil {
			return fmt.Errorf("failed to generate tokens: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func (s *authService) Login(ctx context.Context, cmd service.LoginCommand) (*service.AuthTokens, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, cmd.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// Verify password
	err = auth.ComparePassword(user.PasswordHash, cmd.Password)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	var tokens *service.AuthTokens
	err = s.txManager.ReadCommitted(ctx, func(ctx context.Context) error {
		// Revoke all existing refresh tokens for this user (single session policy)
		err := s.refreshTokenRepo.RevokeAllByUserID(ctx, user.ID)
		if err != nil {
			return fmt.Errorf("failed to revoke existing tokens: %w", err)
		}

		// Generate new tokens
		tokens, err = s.generateTokens(ctx, user)
		if err != nil {
			return fmt.Errorf("failed to generate tokens: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func (s *authService) Refresh(ctx context.Context, cmd service.RefreshTokenCommand) (*service.AuthTokens, error) {
	// Hash the provided token
	tokenHash := auth.HashRefreshToken(cmd.RefreshToken)

	// Get refresh token from database
	refreshToken, err := s.refreshTokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}
	if refreshToken == nil {
		return nil, ErrInvalidRefreshToken
	}

	// Validate token
	if !refreshToken.IsValid() {
		return nil, ErrInvalidRefreshToken
	}

	var tokens *service.AuthTokens
	err = s.txManager.ReadCommitted(ctx, func(ctx context.Context) error {
		// Revoke old token
		refreshToken.Revoke()
		err := s.refreshTokenRepo.Update(ctx, refreshToken)
		if err != nil {
			return fmt.Errorf("failed to revoke old token: %w", err)
		}

		user, err := s.userRepo.GetByID(ctx, refreshToken.UserID)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}
		if user == nil {
			return ErrInvalidRefreshToken
		}

		// Generate new tokens
		tokens, err = s.generateTokens(ctx, user)
		if err != nil {
			return fmt.Errorf("failed to generate new tokens: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func (s *authService) Logout(ctx context.Context, cmd service.LogoutCommand) error {
	// Hash the provided token
	tokenHash := auth.HashRefreshToken(cmd.RefreshToken)

	// Revoke the token
	err := s.refreshTokenRepo.RevokeByTokenHash(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	return nil
}

func (s *authService) generateTokens(ctx context.Context, user *models.User) (*service.AuthTokens, error) {
	// Generate access token
	accessToken, err := s.jwtService.SignAccessToken(user.ID.String(), user.Role, s.accessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token
	refreshTokenStr, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Hash and store refresh token
	tokenHash := auth.HashRefreshToken(refreshTokenStr)
	expiresAt := time.Now().Add(s.refreshTokenTTL)

	refreshTokenModel := models.NewRefreshToken(user.ID, tokenHash, expiresAt)

	_, err = s.refreshTokenRepo.Create(ctx, refreshTokenModel)
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &service.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		ExpiresIn:    int(s.accessTokenTTL.Seconds()),
	}, nil
}
