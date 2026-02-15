package services

import "context"

// Fetcher performs HTTP fetches with configured authentication and retry options
type Fetcher interface {
	Fetch(ctx context.Context, url string) (*FetchResult, error)
}

// FetchResult contains the results of a successful page fetch
type FetchResult struct {
	Body        []byte
	FinalURL    string
	StatusCode  int
	ContentType string
}
