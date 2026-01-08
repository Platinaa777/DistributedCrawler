package valueobjects

type CrawlTaskID struct {
	ID
}

func NewCrawlTaskID(id string) (CrawlTaskID, error) {
	baseID, err := NewID(id)
	if err != nil {
		return CrawlTaskID{}, err
	}
	return CrawlTaskID{ID: baseID}, nil
}

func GenerateCrawlTaskID() CrawlTaskID {
	return CrawlTaskID{ID: GenerateID()}
}
