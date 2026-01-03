package valueobjects

type CrawlJobID struct {
	ID
}

func NewCrawlJobID(id string) (CrawlJobID, error) {
	baseID, err := NewID(id)
	if err != nil {
		return CrawlJobID{}, err
	}
	return CrawlJobID{ID: baseID}, nil
}

func GenerateCrawlJobID() CrawlJobID {
	return CrawlJobID{ID: GenerateID()}
}
