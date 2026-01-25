package models

import (
	"distributed-crawler/internal/domain/crawl/valueobjects"
)

// JobType represents whether a crawl job is one-time or scheduled.
type JobType string

const (
	// JobTypeOnce indicates a one-time crawl job that runs exactly once.
	JobTypeOnce JobType = "ONCE"
	// JobTypeScheduled indicates a recurring crawl job that runs on a schedule.
	JobTypeScheduled JobType = "SCHEDULED"
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

	// JobType determines whether this is a one-time job or a scheduled recurring job.
	// Use JobTypeOnce for exactly-once execution, JobTypeScheduled for recurring jobs.
	JobType JobType

	// RespectRobotsTxt controls whether the crawler follows robots.txt rules.
	// If true, robots.txt rules are fetched and applied to allow/deny URL patterns.
	// If false, robots.txt is ignored and all URLs within scope are crawled.
	RespectRobotsTxt bool
}
