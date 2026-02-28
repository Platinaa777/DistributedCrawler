package models

import (
	"distributed-crawler/internal/domain/crawl/valueobjects"
)

// CrawlMode controls which link-following strategy the crawler uses.
type CrawlMode string

const (
	// CrawlModePaginationAndLinks follows both pagination links and regular <a href> links.
	// This is the default behavior when crawl_mode is not set.
	CrawlModePaginationAndLinks CrawlMode = "pagination_and_links"
	// CrawlModePaginationOnly follows only pagination links defined in ExtractionSpec.Pagination.
	// Regular <a href> link discovery is disabled.
	CrawlModePaginationOnly CrawlMode = "pagination_only"
	// CrawlModeLinksOnly follows only regular <a href> links within scope.
	// Pagination logic is completely ignored.
	CrawlModeLinksOnly CrawlMode = "links_only"
)

// JobType represents whether a crawl job is one-time or scheduled.
type JobType string

const (
	// JobTypeOnce indicates a one-time crawl job that runs exactly once.
	JobTypeOnce JobType = "JOB_TYPE_ONCE"
	// JobTypeScheduled indicates a recurring crawl job that runs on a schedule.
	JobTypeScheduled JobType = "JOB_TYPE_SCHEDULED"
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

	// CrawlMode controls link-following behavior.
	// Use CrawlModePaginationOnly, CrawlModeLinksOnly, or CrawlModePaginationAndLinks.
	// Defaults to CrawlModePaginationAndLinks when empty.
	CrawlMode CrawlMode

	// QueueEndpointAssignments lists the queue endpoints assigned to this job config with routing weights.
	QueueEndpointAssignments []QueueEndpointAssignment
}

// QueueEndpointAssignment links a queue endpoint to a job config with a routing weight.
type QueueEndpointAssignment struct {
	EndpointID string
	Weight     int32
}
