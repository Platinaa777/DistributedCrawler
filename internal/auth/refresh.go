package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

const refreshTokenLength = 32 // 32 bytes = 256 bits

var (
	ErrTokenGeneration = errors.New("failed to generate token")
)

// GenerateRefreshToken generates a cryptographically secure random refresh token
func GenerateRefreshToken() (string, error) {
	b := make([]byte, refreshTokenLength)
	_, err := rand.Read(b)
	if err != nil {
		return "", ErrTokenGeneration
	}

	return hex.EncodeToString(b), nil
}

// HashRefreshToken hashes a refresh token using SHA-256
func HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
