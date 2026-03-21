package auth

import (
	"context"
	"testing"
	"time"

	authmodels "distributed-crawler/internal/domain/auth/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestPasswordAndRefreshTokenHelpers(t *testing.T) {
	t.Parallel()

	hash, err := HashPassword("password123")
	require.NoError(t, err)
	require.NotEmpty(t, hash)
	require.NoError(t, ComparePassword(hash, "password123"))
	require.Error(t, ComparePassword(hash, "wrong"))

	_, err = HashPassword("")
	require.ErrorIs(t, err, ErrInvalidPassword)

	token, err := GenerateRefreshToken()
	require.NoError(t, err)
	assert.Len(t, token, 64)
	assert.NotEqual(t, token, HashRefreshToken(token))
	assert.Equal(t, HashRefreshToken(token), HashRefreshToken(token))
}

func TestJWTService_SignAndVerifyToken(t *testing.T) {
	t.Parallel()

	svc := NewJWTService("secret", "issuer", "audience")
	token, err := svc.SignAccessToken("user-1", authmodels.RoleReadWrite, time.Hour)
	require.NoError(t, err)

	claims, err := svc.VerifyToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, string(authmodels.RoleReadWrite), claims.Role)
}

func TestJWTService_VerifyRejectsExpiredAndInvalidClaims(t *testing.T) {
	t.Parallel()

	svc := NewJWTService("secret", "issuer", "audience")
	expiredToken, err := svc.SignAccessToken("user-1", authmodels.RoleRead, -time.Minute)
	require.NoError(t, err)

	claims, err := svc.VerifyToken(expiredToken)
	require.ErrorIs(t, err, ErrExpiredToken)
	assert.Nil(t, claims)

	invalidRoleToken, err := NewJWTService("secret", "issuer", "audience").SignAccessToken("user-1", authmodels.Role("BAD"), time.Hour)
	require.NoError(t, err)
	claims, err = svc.VerifyToken(invalidRoleToken)
	require.ErrorIs(t, err, ErrMissingClaims)
	assert.Nil(t, claims)
}

func TestJWTAuthInterceptor_HandlesPublicAndPrivateEndpoints(t *testing.T) {
	t.Parallel()

	jwtSvc := NewJWTService("secret", "issuer", "audience")
	token, err := jwtSvc.SignAccessToken("user-1", authmodels.RoleReadWrite, time.Hour)
	require.NoError(t, err)

	interceptor := JWTAuthInterceptor(jwtSvc)
	handler := func(ctx context.Context, req any) (any, error) {
		userID, ok := GetUserIDFromContext(ctx)
		require.True(t, ok)
		role, ok := GetUserRoleFromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, "user-1", userID)
		assert.Equal(t, string(authmodels.RoleReadWrite), role)
		return "ok", nil
	}

	resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{
		FullMethod: "/crawler.v1.AuthService/Login",
	}, func(ctx context.Context, req any) (any, error) {
		return "public", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "public", resp)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
	resp, err = interceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: "/crawler.v1.CrawlerService/CreateJob",
	}, handler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)

	resp, err = interceptor(context.Background(), nil, &grpc.UnaryServerInfo{
		FullMethod: "/crawler.v1.CrawlerService/CreateJob",
	}, handler)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestRBACInterceptor_EnforcesRoles(t *testing.T) {
	t.Parallel()

	interceptor := RBACInterceptor()

	ctx := context.WithValue(context.Background(), UserRoleContextKey, string(authmodels.RoleRead))
	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: "/crawler.v1.CrawlerService/ListJobs",
	}, func(ctx context.Context, req any) (any, error) { return "ok", nil })
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)

	resp, err = interceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: "/crawler.v1.UserService/ListUsers",
	}, func(ctx context.Context, req any) (any, error) { return "ok", nil })
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	resp, err = interceptor(context.Background(), nil, &grpc.UnaryServerInfo{
		FullMethod: "/crawler.v1.CrawlerService/ListJobs",
	}, func(ctx context.Context, req any) (any, error) { return "ok", nil })
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestMethodHelpers(t *testing.T) {
	t.Parallel()

	assert.True(t, isPublicEndpoint("/crawler.v1.AuthService/Register"))
	assert.False(t, isPublicEndpoint("/crawler.v1.CrawlerService/ListJobs"))

	token, err := extractTokenFromMetadata(metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer token")))
	require.NoError(t, err)
	assert.Equal(t, "token", token)

	_, err = extractTokenFromMetadata(metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Basic token")))
	require.Error(t, err)

	role, ok := requiredRoleForMethod("/crawler.v1.QueueAdminService/ListQueueEndpoints")
	assert.True(t, ok)
	assert.Equal(t, authmodels.RoleAdministrator, role)
}

