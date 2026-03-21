package models

import (
	"testing"
	"time"

	authvalueobjects "distributed-crawler/internal/domain/auth/valueobjects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRole_ParseValidateAndLevel(t *testing.T) {
	t.Parallel()

	role, err := ParseRole("READ_WRITE")
	require.NoError(t, err)
	assert.Equal(t, RoleReadWrite, role)
	assert.True(t, RoleAdministrator.IsValid())
	assert.False(t, Role("BAD").IsValid())
	assert.Equal(t, 3, RoleAdministrator.Level())
	assert.Equal(t, 0, Role("BAD").Level())

	_, err = ParseRole("BAD")
	require.Error(t, err)
}

func TestUserAndRefreshTokenHelpers(t *testing.T) {
	t.Parallel()

	user := NewUser("user@example.com", "hash")
	assert.Equal(t, RoleRead, user.Role)
	assert.Equal(t, "user@example.com", user.Email)

	admin := NewUserWithRole("admin@example.com", "hash", RoleAdministrator)
	assert.Equal(t, RoleAdministrator, admin.Role)

	token := NewRefreshToken(authvalueobjects.GenerateUserID(), "hash", time.Now().Add(time.Hour))
	assert.False(t, token.IsRevoked())
	assert.False(t, token.IsExpired())
	assert.True(t, token.IsValid())

	token.Revoke()
	assert.True(t, token.IsRevoked())
	assert.False(t, token.IsValid())
}

