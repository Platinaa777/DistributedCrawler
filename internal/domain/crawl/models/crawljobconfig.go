package models

import (
	"distributed-crawler/internal/domain/crawl/valueobjects"
)

type CrawlJobConfig struct {
	ID             valueobjects.ID
	Name           string
	ExtractionSpec ExtractionSpec
	Scopes         ScopeRules
	Seeds          []Seed
	RateLimit      RateLimitPolicy
	Retries        RetryPolicy
	Auth           AuthOptions
	Schedule       ScheduleOptions
}
