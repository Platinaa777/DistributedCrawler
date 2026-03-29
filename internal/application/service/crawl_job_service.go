package service

import (
	"context"
	"time"

	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
)

// Commands for CrawlJob management

type CreateCrawlJobCommand struct {
	Config          models.CrawlJobConfig
	UserID          string
	AvailableQueues []string // all crawl queue names known to the API server
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

// JobSortField defines which job field to sort by
type JobSortField string

const (
	JobSortByCreatedAt JobSortField = "created_at" // default
	JobSortByName      JobSortField = "name"
	JobSortByStatus    JobSortField = "status"
)

// ListCrawlJobsFilter contains filter criteria for listing jobs
type ListCrawlJobsFilter struct {
	Name        *string    // Partial match on config name (ILIKE %name%)
	UserEmail   *string    // Partial match on creator email (ILIKE %email%)
	Status      *string    // Exact match on job status
	CreatedFrom *time.Time // Jobs created >= this timestamp
	CreatedTo   *time.Time // Jobs created <= this timestamp
}

// ListCrawlJobsCursor represents decoded pagination cursor
type ListCrawlJobsCursor struct {
	SortField string    `json:"sf,omitempty"` // sort field at time of pagination
	SortAsc   bool      `json:"sa"`           // sort direction at time of pagination
	CreatedAt time.Time `json:"c"`            // always set
	Name      string    `json:"n,omitempty"`  // set when sorting by name
	Status    string    `json:"st,omitempty"` // set when sorting by status
	ID        string    `json:"i"`
}

// ListCrawlJobsQuery contains pagination and filter parameters
type ListCrawlJobsQuery struct {
	Cursor    *ListCrawlJobsCursor // nil for first page
	Limit     int                  // Default: 20, Max: 100
	Filter    ListCrawlJobsFilter
	SortField JobSortField // field to sort by
	SortAsc   bool         // true = ASC, false = DESC
}

// ListCrawlJobsResult contains paginated results
type ListCrawlJobsResult struct {
	Jobs       []*models.CrawlJob
	NextCursor *ListCrawlJobsCursor // nil if no more results
	HasMore    bool
}

// Service interfaces

type DeleteCrawlJobCommand struct {
	JobID string
}

type CrawlJobService interface {
	CreateCrawlJob(ctx context.Context, cmd CreateCrawlJobCommand) (valueobjects.CrawlJobID, error)
	GetCrawlJob(ctx context.Context, query GetCrawlJobQuery) (*models.CrawlJob, error)
	ListCrawlJobs(ctx context.Context, query ListCrawlJobsQuery) (*ListCrawlJobsResult, error)
	DeleteCrawlJob(ctx context.Context, cmd DeleteCrawlJobCommand) error
}
