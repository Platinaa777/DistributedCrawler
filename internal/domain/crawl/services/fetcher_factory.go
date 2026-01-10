package services

import "distributed-crawler/internal/domain/crawl/models"

// FetcherFactory creates Fetcher instances configured with task-specific options
type FetcherFactory interface {
	CreateFetcher(auth models.AuthOptions, retry models.RetryPolicy) Fetcher
}
