package service

import (
	"context"
	"time"

	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
)

// Commands for CrawlJob management

type CreateCrawlJobCommand struct {
	Config models.CrawlJobConfig
}

type UpdateCrawlJobStatusCommand struct {
	JobID  string
	Status string
}

type CompleteCrawlJobCommand struct {
	JobID string
}

// Queries for CrawlJob

type GetCrawlJobQuery struct {
	ID string
}

// ListCrawlJobsFilter contains filter criteria for listing jobs
type ListCrawlJobsFilter struct {
	Name        *string    // Partial match on config name (ILIKE %name%)
	Status      *string    // Exact match on job status
	CreatedFrom *time.Time // Jobs created >= this timestamp
	CreatedTo   *time.Time // Jobs created <= this timestamp
}

// ListCrawlJobsCursor represents decoded pagination cursor
type ListCrawlJobsCursor struct {
	CreatedAt time.Time `json:"c"`
	ID        string    `json:"i"`
}

// ListCrawlJobsQuery contains pagination and filter parameters
type ListCrawlJobsQuery struct {
	Cursor *ListCrawlJobsCursor // nil for first page
	Limit  int                  // Default: 20, Max: 100
	Filter ListCrawlJobsFilter
}

// ListCrawlJobsResult contains paginated results
type ListCrawlJobsResult struct {
	Jobs       []*models.CrawlJob
	NextCursor *ListCrawlJobsCursor // nil if no more results
	HasMore    bool
}

// Service interfaces

type CrawlJobService interface {
	CreateCrawlJob(ctx context.Context, cmd CreateCrawlJobCommand) (valueobjects.CrawlJobID, error)
	GetCrawlJob(ctx context.Context, query GetCrawlJobQuery) (*models.CrawlJob, error)
	ListCrawlJobs(ctx context.Context, query ListCrawlJobsQuery) (*ListCrawlJobsResult, error)
}
