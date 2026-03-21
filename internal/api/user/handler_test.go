package user

import (
	"context"
	"testing"
	"time"

	"distributed-crawler/internal/application/service"
	userservice "distributed-crawler/internal/application/service/user"
	authmodels "distributed-crawler/internal/domain/auth/models"
	authvalueobjects "distributed-crawler/internal/domain/auth/valueobjects"
	crawlergrpc "distributed-crawler/pkg/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeUserService struct {
	listFn   func(ctx context.Context) ([]*authmodels.User, error)
	updateFn func(ctx context.Context, cmd service.UpdateUserRoleCommand) error
}

func (f fakeUserService) ListUsers(ctx context.Context) ([]*authmodels.User, error) {
	return f.listFn(ctx)
}
func (f fakeUserService) UpdateUserRole(ctx context.Context, cmd service.UpdateUserRoleCommand) error {
	return f.updateFn(ctx, cmd)
}

func TestListUsers_ConvertsDomainUsers(t *testing.T) {
	t.Parallel()

	userID := authvalueobjects.GenerateUserID()
	createdAt := time.Now().UTC().Round(0)
	updatedAt := createdAt.Add(time.Minute)

	impl := NewImplementation(fakeUserService{
		listFn: func(ctx context.Context) ([]*authmodels.User, error) {
			return []*authmodels.User{{
				ID:        userID,
				Email:     "user@example.com",
				Role:      authmodels.RoleAdministrator,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			}}, nil
		},
	})

	resp, err := impl.ListUsers(context.Background(), &crawlergrpc.ListUsersRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Users, 1)
	assert.Equal(t, userID.String(), resp.Users[0].Id)
	assert.Equal(t, crawlergrpc.Role_ROLE_ADMINISTRATOR, resp.Users[0].Role)
	assert.True(t, resp.Users[0].CreatedAt.AsTime().Equal(createdAt))
}

func TestUpdateUserRole_MapsKnownErrorsToStatusCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{name: "invalid role", err: userservice.ErrInvalidRole, code: codes.InvalidArgument},
		{name: "invalid user id", err: userservice.ErrInvalidUserID, code: codes.InvalidArgument},
		{name: "not found", err: userservice.ErrUserNotFound, code: codes.NotFound},
		{name: "forbidden", err: userservice.ErrRoleNotAllowed, code: codes.PermissionDenied},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			impl := NewImplementation(fakeUserService{
				updateFn: func(ctx context.Context, cmd service.UpdateUserRoleCommand) error {
					return tt.err
				},
			})

			resp, err := impl.UpdateUserRole(context.Background(), &crawlergrpc.UpdateUserRoleRequest{
				Id:   "user-id",
				Role: crawlergrpc.Role_ROLE_ADMINISTRATOR,
			})
			require.Error(t, err)
			assert.Nil(t, resp)
			assert.Equal(t, tt.code, status.Code(err))
		})
	}
}

func TestUpdateUserRole_ReturnsUpdatedFalseWhenRoleUnchanged(t *testing.T) {
	t.Parallel()

	impl := NewImplementation(fakeUserService{
		updateFn: func(ctx context.Context, cmd service.UpdateUserRoleCommand) error {
			assert.Equal(t, "user-id", cmd.UserID)
			assert.Equal(t, authmodels.RoleReadWrite, cmd.Role)
			return userservice.ErrRoleUnchanged
		},
	})

	resp, err := impl.UpdateUserRole(context.Background(), &crawlergrpc.UpdateUserRoleRequest{
		Id:   "user-id",
		Role: crawlergrpc.Role_ROLE_READ_WRITE,
	})
	require.NoError(t, err)
	assert.False(t, resp.Updated)
}

func TestUpdateUserRole_RejectsMissingIDAndUnknownRole(t *testing.T) {
	t.Parallel()

	impl := NewImplementation(fakeUserService{
		updateFn: func(ctx context.Context, cmd service.UpdateUserRoleCommand) error { return nil },
	})

	resp, err := impl.UpdateUserRole(context.Background(), &crawlergrpc.UpdateUserRoleRequest{
		Role: crawlergrpc.Role_ROLE_READ,
	})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	resp, err = impl.UpdateUserRole(context.Background(), &crawlergrpc.UpdateUserRoleRequest{
		Id:   "user-id",
		Role: crawlergrpc.Role_ROLE_UNSPECIFIED,
	})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

