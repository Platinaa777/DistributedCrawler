package valueobjects

import "github.com/google/uuid"

type OutboxEventID struct {
	value string
}

func NewOutboxEventID(value string) (OutboxEventID, error) {
	if value == "" {
		return OutboxEventID{}, ErrEmptyID
	}
	return OutboxEventID{value: value}, nil
}

func GenerateOutboxEventID() OutboxEventID {
	return OutboxEventID{value: uuid.New().String()}
}

func (id OutboxEventID) String() string {
	return id.value
}
