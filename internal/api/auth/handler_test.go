package auth

import (
	"context"
	"errors"
	"testing"

	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAuthService struct {
	registerFn func(ctx context.Context, cmd service.RegisterCommand) (*service.AuthTokens, error)
	loginFn    func(ctx context.Context, cmd service.LoginCommand) (*service.AuthTokens, error)
	refreshFn  func(ctx context.Context, cmd service.RefreshTokenCommand) (*service.AuthTokens, error)
	logoutFn   func(ctx context.Context, cmd service.LogoutCommand) error
}

func (f fakeAuthService) Register(ctx context.Context, cmd service.RegisterCommand) (*service.AuthTokens, error) {
	return f.registerFn(ctx, cmd)
}
func (f fakeAuthService) Login(ctx context.Context, cmd service.LoginCommand) (*service.AuthTokens, error) {
	return f.loginFn(ctx, cmd)
}
func (f fakeAuthService) Refresh(ctx context.Context, cmd service.RefreshTokenCommand) (*service.AuthTokens, error) {
	return f.refreshFn(ctx, cmd)
}
func (f fakeAuthService) Logout(ctx context.Context, cmd service.LogoutCommand) error {
	return f.logoutFn(ctx, cmd)
}

func TestRegister_MapsCommandAndResponse(t *testing.T) {
	t.Parallel()

	impl := NewImplementation(fakeAuthService{
		registerFn: func(ctx context.Context, cmd service.RegisterCommand) (*service.AuthTokens, error) {
			assert.Equal(t, "user@example.com", cmd.Email)
			assert.Equal(t, "secret", cmd.Password)
			return &service.AuthTokens{AccessToken: "access", RefreshToken: "refresh", ExpiresIn: 3600}, nil
		},
	})

	resp, err := impl.Register(context.Background(), &crawlergrpc.RegisterRequest{
		Email:    "user@example.com",
		Password: "secret",
	})
	require.NoError(t, err)
	assert.Equal(t, "access", resp.AccessToken)
	assert.Equal(t, "refresh", resp.RefreshToken)
	assert.Equal(t, int32(3600), resp.ExpiresIn)
}

func TestLogin_PropagatesServiceError(t *testing.T) {
	t.Parallel()

	impl := NewImplementation(fakeAuthService{
		loginFn: func(ctx context.Context, cmd service.LoginCommand) (*service.AuthTokens, error) {
			assert.Equal(t, "user@example.com", cmd.Email)
			return nil, errors.New("bad credentials")
		},
	})

	resp, err := impl.Login(context.Background(), &crawlergrpc.LoginRequest{
		Email:    "user@example.com",
		Password: "wrong",
	})
	require.Error(t, err)
	assert.Nil(t, resp)
}

func TestRefresh_MapsCommandAndResponse(t *testing.T) {
	t.Parallel()

	impl := NewImplementation(fakeAuthService{
		refreshFn: func(ctx context.Context, cmd service.RefreshTokenCommand) (*service.AuthTokens, error) {
			assert.Equal(t, "refresh-token", cmd.RefreshToken)
			return &service.AuthTokens{AccessToken: "new-access", RefreshToken: "new-refresh", ExpiresIn: 1200}, nil
		},
	})

	resp, err := impl.Refresh(context.Background(), &crawlergrpc.RefreshRequest{RefreshToken: "refresh-token"})
	require.NoError(t, err)
	assert.Equal(t, "new-access", resp.AccessToken)
	assert.Equal(t, "new-refresh", resp.RefreshToken)
	assert.Equal(t, int32(1200), resp.ExpiresIn)
}

func TestLogout_MapsCommandAndReturnsEmptyResponse(t *testing.T) {
	t.Parallel()

	impl := NewImplementation(fakeAuthService{
		logoutFn: func(ctx context.Context, cmd service.LogoutCommand) error {
			assert.Equal(t, "refresh-token", cmd.RefreshToken)
			return nil
		},
	})

	resp, err := impl.Logout(context.Background(), &crawlergrpc.LogoutRequest{RefreshToken: "refresh-token"})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

