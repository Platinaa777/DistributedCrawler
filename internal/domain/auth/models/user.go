package models

import (
	"distributed-crawler/internal/domain/auth/valueobjects"
	"time"
)

type User struct {
	ID           valueobjects.UserID
	Email        string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewUser(email, passwordHash string) *User {
	now := time.Now()
	return &User{
		ID:           valueobjects.GenerateUserID(),
		Email:        email,
		PasswordHash: passwordHash,
		Role:         RoleRead,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func NewUserWithRole(email, passwordHash string, role Role) *User {
	now := time.Now()
	return &User{
		ID:           valueobjects.GenerateUserID(),
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
