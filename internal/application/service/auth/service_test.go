package auth

import (
	"context"
	"testing"
	"time"

	"distributed-crawler/internal/application/service"
	coreauth "distributed-crawler/internal/auth"
	authmodels "distributed-crawler/internal/domain/auth/models"
	refreshtokenrepo "distributed-crawler/internal/domain/auth/repos/refresh_token"
	userrepo "distributed-crawler/internal/domain/auth/repos/user"
	authvalueobjects "distributed-crawler/internal/domain/auth/valueobjects"
	"distributed-crawler/internal/infra/persistence"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type authUserRepoFake struct {
	getByEmailFn func(ctx context.Context, email string) (*authmodels.User, error)
	createFn     func(ctx context.Context, entity *authmodels.User) (authvalueobjects.UserID, error)
	getByIDFn    func(ctx context.Context, id authvalueobjects.UserID) (*authmodels.User, error)
}

func (f authUserRepoFake) Create(ctx context.Context, entity *authmodels.User) (authvalueobjects.UserID, error) {
	return f.createFn(ctx, entity)
}
func (f authUserRepoFake) GetByID(ctx context.Context, id authvalueobjects.UserID) (*authmodels.User, error) {
	return f.getByIDFn(ctx, id)
}
func (f authUserRepoFake) GetByEmail(ctx context.Context, email string) (*authmodels.User, error) {
	return f.getByEmailFn(ctx, email)
}
func (f authUserRepoFake) Update(context.Context, *authmodels.User) error { return nil }
func (f authUserRepoFake) List(context.Context) ([]*authmodels.User, error) { return nil, nil }
func (f authUserRepoFake) UpdateRole(context.Context, authvalueobjects.UserID, authmodels.Role) error {
	return nil
}

type refreshTokenRepoFake struct {
	createFn          func(ctx context.Context, entity *authmodels.RefreshToken) (authvalueobjects.RefreshTokenID, error)
	getByHashFn       func(ctx context.Context, tokenHash string) (*authmodels.RefreshToken, error)
	updateFn          func(ctx context.Context, entity *authmodels.RefreshToken) error
	revokeByHashFn    func(ctx context.Context, tokenHash string) error
	revokeAllByUserFn func(ctx context.Context, userID authvalueobjects.UserID) error
}

func (f refreshTokenRepoFake) Create(ctx context.Context, entity *authmodels.RefreshToken) (authvalueobjects.RefreshTokenID, error) {
	return f.createFn(ctx, entity)
}
func (f refreshTokenRepoFake) GetByTokenHash(ctx context.Context, tokenHash string) (*authmodels.RefreshToken, error) {
	return f.getByHashFn(ctx, tokenHash)
}
func (f refreshTokenRepoFake) GetByID(context.Context, authvalueobjects.RefreshTokenID) (*authmodels.RefreshToken, error) {
	return nil, nil
}
func (f refreshTokenRepoFake) Update(ctx context.Context, entity *authmodels.RefreshToken) error {
	return f.updateFn(ctx, entity)
}
func (f refreshTokenRepoFake) RevokeByTokenHash(ctx context.Context, tokenHash string) error {
	return f.revokeByHashFn(ctx, tokenHash)
}
func (f refreshTokenRepoFake) RevokeAllByUserID(ctx context.Context, userID authvalueobjects.UserID) error {
	return f.revokeAllByUserFn(ctx, userID)
}

type txManagerFake struct {
	runFn func(ctx context.Context, exec persistence.Handler) error
}

func (f txManagerFake) ReadCommitted(ctx context.Context, exec persistence.Handler) error {
	return f.runFn(ctx, exec)
}

var _ userrepo.UserRepository = authUserRepoFake{}
var _ refreshtokenrepo.RefreshTokenRepository = refreshTokenRepoFake{}

func TestRegister_ValidatesInputsAndCreatesTokens(t *testing.T) {
	t.Parallel()

	jwtSvc := coreauth.NewJWTService("secret", "issuer", "audience")
	svc := NewAuthService(
		authUserRepoFake{
			getByEmailFn: func(ctx context.Context, email string) (*authmodels.User, error) {
				assert.Equal(t, "user@example.com", email)
				return nil, nil
			},
			createFn: func(ctx context.Context, entity *authmodels.User) (authvalueobjects.UserID, error) {
				assert.Equal(t, "user@example.com", entity.Email)
				assert.Equal(t, authmodels.RoleRead, entity.Role)
				assert.NotEmpty(t, entity.PasswordHash)
				return entity.ID, nil
			},
			getByIDFn: func(ctx context.Context, id authvalueobjects.UserID) (*authmodels.User, error) { return nil, nil },
		},
		refreshTokenRepoFake{
			createFn: func(ctx context.Context, entity *authmodels.RefreshToken) (authvalueobjects.RefreshTokenID, error) {
				assert.NotEmpty(t, entity.TokenHash)
				return entity.ID, nil
			},
			getByHashFn:       func(ctx context.Context, tokenHash string) (*authmodels.RefreshToken, error) { return nil, nil },
			updateFn:          func(ctx context.Context, entity *authmodels.RefreshToken) error { return nil },
			revokeByHashFn:    func(ctx context.Context, tokenHash string) error { return nil },
			revokeAllByUserFn: func(ctx context.Context, userID authvalueobjects.UserID) error { return nil },
		},
		txManagerFake{runFn: func(ctx context.Context, exec persistence.Handler) error { return exec(ctx) }},
		jwtSvc,
		time.Hour,
		24*time.Hour,
	)

	tokens, err := svc.Register(context.Background(), service.RegisterCommand{
		Email:    "user@example.com",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	assert.Equal(t, 3600, tokens.ExpiresIn)

	_, err = svc.Register(context.Background(), service.RegisterCommand{Email: "bad", Password: "password123"})
	require.ErrorIs(t, err, ErrInvalidEmail)

	_, err = svc.Register(context.Background(), service.RegisterCommand{Email: "user@example.com", Password: "short"})
	require.ErrorIs(t, err, ErrPasswordTooShort)
}

func TestRegister_RejectsExistingUser(t *testing.T) {
	t.Parallel()

	svc := NewAuthService(
		authUserRepoFake{
			getByEmailFn: func(ctx context.Context, email string) (*authmodels.User, error) {
				return &authmodels.User{Email: email}, nil
			},
			createFn:  func(ctx context.Context, entity *authmodels.User) (authvalueobjects.UserID, error) { return authvalueobjects.UserID{}, nil },
			getByIDFn: func(ctx context.Context, id authvalueobjects.UserID) (*authmodels.User, error) { return nil, nil },
		},
		refreshTokenRepoFake{
			createFn:          func(ctx context.Context, entity *authmodels.RefreshToken) (authvalueobjects.RefreshTokenID, error) { return authvalueobjects.RefreshTokenID{}, nil },
			getByHashFn:       func(ctx context.Context, tokenHash string) (*authmodels.RefreshToken, error) { return nil, nil },
			updateFn:          func(ctx context.Context, entity *authmodels.RefreshToken) error { return nil },
			revokeByHashFn:    func(ctx context.Context, tokenHash string) error { return nil },
			revokeAllByUserFn: func(ctx context.Context, userID authvalueobjects.UserID) error { return nil },
		},
		txManagerFake{runFn: func(ctx context.Context, exec persistence.Handler) error { return exec(ctx) }},
		coreauth.NewJWTService("secret", "issuer", "audience"),
		time.Hour,
		24*time.Hour,
	)

	_, err := svc.Register(context.Background(), service.RegisterCommand{Email: "user@example.com", Password: "password123"})
	require.ErrorIs(t, err, ErrUserExists)
}

func TestLoginRefreshAndLogout(t *testing.T) {
	t.Parallel()

	user := authmodels.NewUser("user@example.com", mustHashPassword(t, "password123"))
	jwtSvc := coreauth.NewJWTService("secret", "issuer", "audience")
	var savedRefreshHash string

	svc := NewAuthService(
		authUserRepoFake{
			getByEmailFn: func(ctx context.Context, email string) (*authmodels.User, error) {
				return user, nil
			},
			createFn: func(ctx context.Context, entity *authmodels.User) (authvalueobjects.UserID, error) {
				return entity.ID, nil
			},
			getByIDFn: func(ctx context.Context, id authvalueobjects.UserID) (*authmodels.User, error) {
				assert.Equal(t, user.ID, id)
				return user, nil
			},
		},
		refreshTokenRepoFake{
			createFn: func(ctx context.Context, entity *authmodels.RefreshToken) (authvalueobjects.RefreshTokenID, error) {
				savedRefreshHash = entity.TokenHash
				return entity.ID, nil
			},
			getByHashFn: func(ctx context.Context, tokenHash string) (*authmodels.RefreshToken, error) {
				if tokenHash != savedRefreshHash {
					return nil, nil
				}
				return authmodels.NewRefreshToken(user.ID, tokenHash, time.Now().Add(time.Hour)), nil
			},
			updateFn: func(ctx context.Context, entity *authmodels.RefreshToken) error {
				assert.True(t, entity.IsRevoked())
				return nil
			},
			revokeByHashFn: func(ctx context.Context, tokenHash string) error {
				assert.NotEmpty(t, tokenHash)
				return nil
			},
			revokeAllByUserFn: func(ctx context.Context, userID authvalueobjects.UserID) error {
				assert.Equal(t, user.ID, userID)
				return nil
			},
		},
		txManagerFake{runFn: func(ctx context.Context, exec persistence.Handler) error { return exec(ctx) }},
		jwtSvc,
		time.Hour,
		24*time.Hour,
	)

	tokens, err := svc.Login(context.Background(), service.LoginCommand{
		Email:    "user@example.com",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, tokens.RefreshToken)

	refreshed, err := svc.Refresh(context.Background(), service.RefreshTokenCommand{
		RefreshToken: tokens.RefreshToken,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, refreshed.AccessToken)

	err = svc.Logout(context.Background(), service.LogoutCommand{RefreshToken: tokens.RefreshToken})
	require.NoError(t, err)
}

func TestLoginAndRefresh_InvalidCredentialsAndToken(t *testing.T) {
	t.Parallel()

	svc := NewAuthService(
		authUserRepoFake{
			getByEmailFn: func(ctx context.Context, email string) (*authmodels.User, error) { return nil, nil },
			createFn:     func(ctx context.Context, entity *authmodels.User) (authvalueobjects.UserID, error) { return authvalueobjects.UserID{}, nil },
			getByIDFn:    func(ctx context.Context, id authvalueobjects.UserID) (*authmodels.User, error) { return nil, nil },
		},
		refreshTokenRepoFake{
			createFn:          func(ctx context.Context, entity *authmodels.RefreshToken) (authvalueobjects.RefreshTokenID, error) { return authvalueobjects.RefreshTokenID{}, nil },
			getByHashFn:       func(ctx context.Context, tokenHash string) (*authmodels.RefreshToken, error) { return nil, nil },
			updateFn:          func(ctx context.Context, entity *authmodels.RefreshToken) error { return nil },
			revokeByHashFn:    func(ctx context.Context, tokenHash string) error { return nil },
			revokeAllByUserFn: func(ctx context.Context, userID authvalueobjects.UserID) error { return nil },
		},
		txManagerFake{runFn: func(ctx context.Context, exec persistence.Handler) error { return exec(ctx) }},
		coreauth.NewJWTService("secret", "issuer", "audience"),
		time.Hour,
		24*time.Hour,
	)

	_, err := svc.Login(context.Background(), service.LoginCommand{Email: "missing@example.com", Password: "password123"})
	require.ErrorIs(t, err, ErrInvalidCredentials)

	_, err = svc.Refresh(context.Background(), service.RefreshTokenCommand{RefreshToken: "missing"})
	require.ErrorIs(t, err, ErrInvalidRefreshToken)
}

func mustHashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := coreauth.HashPassword(password)
	require.NoError(t, err)
	return hash
}
