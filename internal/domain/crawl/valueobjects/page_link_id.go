package valueobjects

type PageLinkID struct {
	ID
}

func NewPageLinkID(id string) (PageLinkID, error) {
	baseID, err := NewID(id)
	if err != nil {
		return PageLinkID{}, err
	}
	return PageLinkID{ID: baseID}, nil
}

func GeneratePageLinkID() PageLinkID {
	return PageLinkID{ID: GenerateID()}
}
