package models

import (
	"distributed-crawler/internal/domain/auth/valueobjects"
	"time"
)

type User struct {
	ID           valueobjects.UserID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewUser(email, passwordHash string) *User {
	now := time.Now()
	return &User{
		ID:           valueobjects.GenerateUserID(),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
