package service

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
)

// Commands for CrawlJob management

type CreateCrawlJobCommand struct {
	Name string
	URLs []string
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

type ListCrawlJobsQuery struct {
	Status string
	Limit  int
	Offset int
}

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

type ListTasksByJobQuery struct {
	JobID string
}

// Service interfaces

type CrawlJobService interface {
	CreateCrawlJob(ctx context.Context, cmd CreateCrawlJobCommand) (valueobjects.CrawlJobID, error)
	GetCrawlJob(ctx context.Context, query GetCrawlJobQuery) (*models.CrawlJob, error)
	ListCrawlJobs(ctx context.Context, query ListCrawlJobsQuery) ([]*models.CrawlJob, error)
	UpdateJobStatus(ctx context.Context, cmd UpdateCrawlJobStatusCommand) error
	CompleteJob(ctx context.Context, cmd CompleteCrawlJobCommand) error
}

type CrawlTaskService interface {
	CreateTask(ctx context.Context, cmd CreateCrawlTaskCommand) (valueobjects.CrawlTaskID, error)
	GetTask(ctx context.Context, query GetCrawlTaskQuery) (*models.CrawlTask, error)
	ListTasksByJob(ctx context.Context, query ListTasksByJobQuery) ([]*models.CrawlTask, error)
	UpdateTaskStatus(ctx context.Context, cmd UpdateTaskStatusCommand) error
}
