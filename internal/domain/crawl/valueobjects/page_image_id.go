package valueobjects

type PageImageID struct {
	ID
}

func NewPageImageID(id string) (PageImageID, error) {
	baseID, err := NewID(id)
	if err != nil {
		return PageImageID{}, err
	}
	return PageImageID{ID: baseID}, nil
}

func GeneratePageImageID() PageImageID {
	return PageImageID{ID: GenerateID()}
}
