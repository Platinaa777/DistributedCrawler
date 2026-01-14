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

	// RespectRobotsTxt controls whether the crawler follows robots.txt rules.
	// If true, robots.txt rules are fetched and applied to allow/deny URL patterns.
	// If false, robots.txt is ignored and all URLs within scope are crawled.
	RespectRobotsTxt bool
}
