package auth

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

var (
	ErrInvalidPassword = errors.New("invalid password")
)

// HashPassword hashes a password using bcrypt with cost 12
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrInvalidPassword
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}

// ComparePassword compares a password with a hash
func ComparePassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
