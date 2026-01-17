package converters

import (
	"distributed-crawler/internal/domain/auth/models"
	"distributed-crawler/internal/domain/auth/valueobjects"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"
)

func UserSnapshotToModel(snapshot snapshots.UserSnapshot) (*models.User, error) {
	userID, err := valueobjects.NewUserID(snapshot.ID)
	if err != nil {
		return nil, err
	}

	role, err := models.ParseRole(snapshot.Role)
	if err != nil {
		return nil, err
	}

	return &models.User{
		ID:           userID,
		Email:        snapshot.Email,
		PasswordHash: snapshot.PasswordHash,
		Role:         role,
		CreatedAt:    snapshot.CreatedAt,
		UpdatedAt:    snapshot.UpdatedAt,
	}, nil
}

func UserModelToSnapshot(user *models.User) snapshots.UserSnapshot {
	return snapshots.UserSnapshot{
		ID:           user.ID.String(),
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Role:         string(user.Role),
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
}
