package valueobjects

type PreviewID struct {
	ID
}

func NewPreviewID(id string) (PreviewID, error) {
	baseID, err := NewID(id)
	if err != nil {
		return PreviewID{}, err
	}
	return PreviewID{ID: baseID}, nil
}

func GeneratePreviewID() PreviewID {
	return PreviewID{ID: GenerateID()}
}
