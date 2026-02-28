package service

import (
	"context"
	"time"

	"distributed-crawler/internal/domain/crawl/models"
)

// Commands for CrawlTask management

type CreateCrawlTaskCommand struct {
	JobID string
	URL   string
}

type UpdateTaskStatusCommand struct {
	TaskID string
	Status string
}

// Queries for CrawlTask

type GetCrawlTaskQuery struct {
	ID string
}

// TaskSortField defines which task field to sort by
type TaskSortField string

const (
	TaskSortByEnqueuedAt TaskSortField = "enqueued_at" // default
	TaskSortByURL        TaskSortField = "url"
	TaskSortByStatus     TaskSortField = "status"
	TaskSortByDepth      TaskSortField = "depth"
)

// ListTasksFilter contains filter criteria for listing tasks
type ListTasksFilter struct {
	Status       *string    // Exact match on task status
	URL          *string    // Partial match on URL (ILIKE %url%)
	MinDepth     *uint64    // Tasks with depth >= this value
	MaxDepth     *uint64    // Tasks with depth <= this value
	EnqueuedFrom *time.Time // Tasks enqueued >= this timestamp
	EnqueuedTo   *time.Time // Tasks enqueued <= this timestamp
}

// ListTasksCursor represents decoded pagination cursor
type ListTasksCursor struct {
	SortField  string     `json:"sf,omitempty"` // sort field at time of pagination
	SortAsc    bool       `json:"sa"`           // sort direction at time of pagination
	EnqueuedAt time.Time  `json:"e"`            // always set
	URL        string     `json:"u,omitempty"`  // set when sorting by url
	Status     string     `json:"st,omitempty"` // set when sorting by status
	Depth      *uint64    `json:"d,omitempty"`  // set when sorting by depth
	ID         string     `json:"i"`
}

// ListTasksByJobQuery contains pagination and filter parameters
type ListTasksByJobQuery struct {
	JobID     string
	Cursor    *ListTasksCursor // nil for first page
	Limit     int              // Default: 20, Max: 100
	Filter    ListTasksFilter
	SortField TaskSortField // field to sort by
	SortAsc   bool          // true = ASC, false = DESC
}

// ListTasksResult contains paginated results
type ListTasksResult struct {
	Tasks      []*models.CrawlTask
	NextCursor *ListTasksCursor // nil if no more results
	HasMore    bool
}

// TaskAnalytics contains aggregated task statistics
type TaskAnalytics struct {
	StatusCounts map[string]int64 // status -> count
	DepthCounts  map[uint64]int64 // depth -> count
	TotalCount   int64
}

// GetTaskAnalyticsQuery requests analytics for a job
type GetTaskAnalyticsQuery struct {
	JobID string
}

type CrawlTaskService interface {
	GetTask(ctx context.Context, query GetCrawlTaskQuery) (*models.CrawlTask, error)
	ListTasksByJob(ctx context.Context, query ListTasksByJobQuery) (*ListTasksResult, error)
	GetTaskAnalytics(ctx context.Context, query GetTaskAnalyticsQuery) (*TaskAnalytics, error)
}
