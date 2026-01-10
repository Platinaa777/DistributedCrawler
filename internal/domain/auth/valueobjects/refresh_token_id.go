package valueobjects

import "distributed-crawler/internal/domain/crawl/valueobjects"

type RefreshTokenID struct {
	valueobjects.ID
}

func NewRefreshTokenID(id string) (RefreshTokenID, error) {
	baseID, err := valueobjects.NewID(id)
	if err != nil {
		return RefreshTokenID{}, err
	}
	return RefreshTokenID{ID: baseID}, nil
}

func GenerateRefreshTokenID() RefreshTokenID {
	return RefreshTokenID{ID: valueobjects.GenerateID()}
}
