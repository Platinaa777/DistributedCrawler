package valueobjects

type ExtractedRecordID struct {
	ID
}

func NewExtractedRecordID(id string) (ExtractedRecordID, error) {
	baseID, err := NewID(id)
	if err != nil {
		return ExtractedRecordID{}, err
	}
	return ExtractedRecordID{ID: baseID}, nil
}

func GenerateExtractedRecordID() ExtractedRecordID {
	return ExtractedRecordID{ID: GenerateID()}
}
