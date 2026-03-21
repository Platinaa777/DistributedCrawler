package valueobjects

import (
	"testing"

	"distributed-crawler/internal/domain/crawl/valueobjects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserAndRefreshTokenIDs(t *testing.T) {
	t.Parallel()

	userID := GenerateUserID()
	require.NotEmpty(t, userID.String())

	parsedUserID, err := NewUserID(userID.String())
	require.NoError(t, err)
	assert.Equal(t, userID.String(), parsedUserID.String())

	refreshID := GenerateRefreshTokenID()
	require.NotEmpty(t, refreshID.String())

	parsedRefreshID, err := NewRefreshTokenID(refreshID.String())
	require.NoError(t, err)
	assert.Equal(t, refreshID.String(), parsedRefreshID.String())

	_, err = NewUserID("")
	require.ErrorIs(t, err, valueobjects.ErrEmptyID)
}

