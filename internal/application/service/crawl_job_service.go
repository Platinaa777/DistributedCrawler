package service

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
)

type CreateCrawlJobCommand struct {
	Name         string
	Status       string
	URL          string
	MaxDepth     int
	ExtractRules []string
}

type GetCrawlJobQuery struct {
	ID string
}

type CrawlJobService interface {
	CreateCrawlJob(ctx context.Context, cmd CreateCrawlJobCommand) (valueobjects.CrawlJobID, error)
	GetCrawlJob(ctx context.Context, query GetCrawlJobQuery) (*models.CrawlJob, error)
}
