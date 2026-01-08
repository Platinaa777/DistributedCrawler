package valueobjects

type PageSnapshotID struct {
	ID
}

func NewPageSnapshotID(id string) (PageSnapshotID, error) {
	baseID, err := NewID(id)
	if err != nil {
		return PageSnapshotID{}, err
	}
	return PageSnapshotID{ID: baseID}, nil
}

func GeneratePageSnapshotID() PageSnapshotID {
	return PageSnapshotID{ID: GenerateID()}
}
