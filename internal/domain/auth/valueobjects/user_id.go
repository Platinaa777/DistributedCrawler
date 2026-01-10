package valueobjects

import "distributed-crawler/internal/domain/crawl/valueobjects"

type UserID struct {
	valueobjects.ID
}

func NewUserID(id string) (UserID, error) {
	baseID, err := valueobjects.NewID(id)
	if err != nil {
		return UserID{}, err
	}
	return UserID{ID: baseID}, nil
}

func GenerateUserID() UserID {
	return UserID{ID: valueobjects.GenerateID()}
}
