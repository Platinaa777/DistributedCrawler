package fetcher

import (
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/services"
)

// SeleniumFetcherFactory implements the FetcherFactory interface for Selenium-based fetching.
type SeleniumFetcherFactory struct {
	remoteURL string // Selenium WebDriver hub URL, e.g. "http://selenium:4444/wd/hub"
}

// NewSeleniumFetcherFactory creates a new Selenium fetcher factory.
// remoteURL is the WebDriver hub endpoint (e.g. "http://selenium:4444/wd/hub").
func NewSeleniumFetcherFactory(remoteURL string) *SeleniumFetcherFactory {
	return &SeleniumFetcherFactory{remoteURL: remoteURL}
}

// CreateFetcher creates a new SeleniumFetcher instance configured with the specified options.
func (f *SeleniumFetcherFactory) CreateFetcher(auth models.AuthOptions, retry models.RetryPolicy) services.Fetcher {
	return NewSeleniumFetcher(auth, retry, f.remoteURL)
}
