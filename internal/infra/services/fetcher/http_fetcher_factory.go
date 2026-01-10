package fetcher

import (
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/services"
)

// HTTPFetcherFactory implements the FetcherFactory interface
type HTTPFetcherFactory struct{}

// NewHTTPFetcherFactory creates a new HTTP fetcher factory
func NewHTTPFetcherFactory() *HTTPFetcherFactory {
	return &HTTPFetcherFactory{}
}

// CreateFetcher creates a new Fetcher instance configured with the specified options
func (f *HTTPFetcherFactory) CreateFetcher(auth models.AuthOptions, retry models.RetryPolicy) services.Fetcher {
	return NewHTTPFetcher(auth, retry)
}
