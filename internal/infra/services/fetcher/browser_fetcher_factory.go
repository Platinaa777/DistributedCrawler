package fetcher

import (
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/services"
)

// BrowserFetcherFactory implements the FetcherFactory interface for browser fetcher.
type BrowserFetcherFactory struct{}

// NewBrowserFetcherFactory creates a new browser fetcher factory.
func NewBrowserFetcherFactory() *BrowserFetcherFactory {
	return &BrowserFetcherFactory{}
}

// CreateFetcher creates a new BrowserFetcher instance configured with the specified options.
func (f *BrowserFetcherFactory) CreateFetcher(auth models.AuthOptions, retry models.RetryPolicy) services.Fetcher {
	return NewBrowserFetcher(auth, retry)
}
