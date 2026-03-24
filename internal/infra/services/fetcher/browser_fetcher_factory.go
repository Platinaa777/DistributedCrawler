package fetcher

import (
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/services"
)

// BrowserFetcherFactory implements the FetcherFactory interface for browser fetcher.
type BrowserFetcherFactory struct {
	remoteURL string // empty = local Chrome process; non-empty = remote CDP endpoint
}

// NewBrowserFetcherFactory creates a new browser fetcher factory.
// remoteURL is the HTTP endpoint of a remote Chrome instance (e.g. "http://chrome:9222").
// Pass an empty string to spawn a local Chrome process per fetch.
func NewBrowserFetcherFactory(remoteURL string) *BrowserFetcherFactory {
	return &BrowserFetcherFactory{remoteURL: remoteURL}
}

// CreateFetcher creates a new BrowserFetcher instance configured with the specified options.
func (f *BrowserFetcherFactory) CreateFetcher(auth models.AuthOptions, retry models.RetryPolicy) services.Fetcher {
	return NewBrowserFetcher(auth, retry, f.remoteURL)
}
