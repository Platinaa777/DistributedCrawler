package valueobjects

import (
	"errors"
	"github.com/google/uuid"
)

var (
	ErrInvalidID = errors.New("invalid ID")
	ErrEmptyID   = errors.New("ID cannot be empty")
)

// ID represents a base unique identifier using UUID
type ID struct {
	value string
}

// NewID creates a new ID from a string
func NewID(id string) (ID, error) {
	if id == "" {
		return ID{}, ErrEmptyID
	}

	// Validate UUID format
	if _, err := uuid.Parse(id); err != nil {
		return ID{}, ErrInvalidID
	}

	return ID{value: id}, nil
}

// GenerateID generates a new unique ID
func GenerateID() ID {
	return ID{value: uuid.New().String()}
}

// String returns the string representation of the ID
func (id ID) String() string {
	return id.value
}

// Equals checks if two IDs are equal
func (id ID) Equals(other ID) bool {
	return id.value == other.value
}

// IsEmpty checks if the ID is empty
func (id ID) IsEmpty() bool {
	return id.value == ""
}
