package service

import (
	"context"
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

type ListTasksByJobQuery struct {
	JobID string
}

type CrawlTaskService interface {
	GetTask(ctx context.Context, query GetCrawlTaskQuery) (*models.CrawlTask, error)
	ListTasksByJob(ctx context.Context, query ListTasksByJobQuery) ([]*models.CrawlTask, error)
}