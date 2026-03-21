package user

import (
	"context"
	"errors"
	"testing"

	"distributed-crawler/internal/application/service"
	authmodels "distributed-crawler/internal/domain/auth/models"
	userrepo "distributed-crawler/internal/domain/auth/repos/user"
	authvalueobjects "distributed-crawler/internal/domain/auth/valueobjects"
	"distributed-crawler/internal/infra/persistence"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type userRepoFake struct {
	listFn       func(ctx context.Context) ([]*authmodels.User, error)
	getByIDFn    func(ctx context.Context, id authvalueobjects.UserID) (*authmodels.User, error)
	updateRoleFn func(ctx context.Context, id authvalueobjects.UserID, role authmodels.Role) error
}

func (f userRepoFake) Create(context.Context, *authmodels.User) (authvalueobjects.UserID, error) {
	return authvalueobjects.UserID{}, nil
}
func (f userRepoFake) GetByID(ctx context.Context, id authvalueobjects.UserID) (*authmodels.User, error) {
	return f.getByIDFn(ctx, id)
}
func (f userRepoFake) GetByEmail(context.Context, string) (*authmodels.User, error) { return nil, nil }
func (f userRepoFake) Update(context.Context, *authmodels.User) error                { return nil }
func (f userRepoFake) List(ctx context.Context) ([]*authmodels.User, error)          { return f.listFn(ctx) }
func (f userRepoFake) UpdateRole(ctx context.Context, id authvalueobjects.UserID, role authmodels.Role) error {
	return f.updateRoleFn(ctx, id, role)
}

type txFake struct {
	runFn func(ctx context.Context, exec persistence.Handler) error
}

func (f txFake) ReadCommitted(ctx context.Context, exec persistence.Handler) error {
	return f.runFn(ctx, exec)
}

var _ userrepo.UserRepository = userRepoFake{}

func TestUserService_ListUsers(t *testing.T) {
	t.Parallel()

	svc := NewUserService(userRepoFake{
		listFn: func(ctx context.Context) ([]*authmodels.User, error) {
			return []*authmodels.User{{Email: "user@example.com"}}, nil
		},
		getByIDFn: func(ctx context.Context, id authvalueobjects.UserID) (*authmodels.User, error) { return nil, nil },
		updateRoleFn: func(ctx context.Context, id authvalueobjects.UserID, role authmodels.Role) error { return nil },
	}, txFake{runFn: func(ctx context.Context, exec persistence.Handler) error { return exec(ctx) }})

	users, err := svc.ListUsers(context.Background())
	require.NoError(t, err)
	require.Len(t, users, 1)
}

func TestUserService_UpdateUserRole_ValidationAndStateChanges(t *testing.T) {
	t.Parallel()

	userID := authvalueobjects.GenerateUserID()
	svc := NewUserService(userRepoFake{
		listFn: func(ctx context.Context) ([]*authmodels.User, error) { return nil, nil },
		getByIDFn: func(ctx context.Context, id authvalueobjects.UserID) (*authmodels.User, error) {
			assert.Equal(t, userID, id)
			return &authmodels.User{ID: userID, Role: authmodels.RoleRead}, nil
		},
		updateRoleFn: func(ctx context.Context, id authvalueobjects.UserID, role authmodels.Role) error {
			assert.Equal(t, authmodels.RoleReadWrite, role)
			return nil
		},
	}, txFake{runFn: func(ctx context.Context, exec persistence.Handler) error { return exec(ctx) }})

	err := svc.UpdateUserRole(context.Background(), service.UpdateUserRoleCommand{UserID: userID.String(), Role: authmodels.RoleReadWrite})
	require.NoError(t, err)

	err = svc.UpdateUserRole(context.Background(), service.UpdateUserRoleCommand{UserID: userID.String(), Role: authmodels.Role("BAD")})
	require.ErrorIs(t, err, ErrInvalidRole)

	err = svc.UpdateUserRole(context.Background(), service.UpdateUserRoleCommand{UserID: userID.String(), Role: authmodels.RoleAdministrator})
	require.ErrorIs(t, err, ErrRoleNotAllowed)

	err = svc.UpdateUserRole(context.Background(), service.UpdateUserRoleCommand{UserID: "bad", Role: authmodels.RoleRead})
	require.ErrorIs(t, err, ErrInvalidUserID)
}

func TestUserService_UpdateUserRole_RepoOutcomes(t *testing.T) {
	t.Parallel()

	userID := authvalueobjects.GenerateUserID()
	tests := []struct {
		name string
		user *authmodels.User
		err  error
		want error
	}{
		{name: "not found", user: nil, want: ErrUserNotFound},
		{name: "unchanged", user: &authmodels.User{ID: userID, Role: authmodels.RoleRead}, want: ErrRoleUnchanged},
		{name: "repo error", err: errors.New("db"), want: errors.New("db")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := NewUserService(userRepoFake{
				listFn: func(ctx context.Context) ([]*authmodels.User, error) { return nil, nil },
				getByIDFn: func(ctx context.Context, id authvalueobjects.UserID) (*authmodels.User, error) {
					return tt.user, tt.err
				},
				updateRoleFn: func(ctx context.Context, id authvalueobjects.UserID, role authmodels.Role) error { return nil },
			}, txFake{runFn: func(ctx context.Context, exec persistence.Handler) error { return exec(ctx) }})

			err := svc.UpdateUserRole(context.Background(), service.UpdateUserRoleCommand{UserID: userID.String(), Role: authmodels.RoleRead})
			require.Error(t, err)
			if tt.name == "repo error" {
				assert.Equal(t, "db", err.Error())
			} else {
				assert.ErrorIs(t, err, tt.want)
			}
		})
	}
}
