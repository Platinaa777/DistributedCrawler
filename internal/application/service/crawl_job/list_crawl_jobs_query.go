package crawljob

import (
	"context"
	"fmt"

	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
)

func (s *crawlJobServ) ListCrawlJobs(ctx context.Context, query service.ListCrawlJobsQuery) (*service.ListCrawlJobsResult, error) {
	// Set defaults
	if query.Limit == 0 {
		query.Limit = 20
	}
	if query.Limit > 100 {
		query.Limit = 100
	}

	// Validate status if provided
	if query.Filter.Status != nil && *query.Filter.Status != "" {
		status := models.TaskStatus(*query.Filter.Status)
		if !status.IsValid() {
			return nil, fmt.Errorf("invalid status: %s", *query.Filter.Status)
		}
	}

	// Validate date range
	if query.Filter.CreatedFrom != nil && query.Filter.CreatedTo != nil {
		if query.Filter.CreatedFrom.After(*query.Filter.CreatedTo) {
			return nil, fmt.Errorf("created_from cannot be after created_to")
		}
	}

	// Call repository with cursor-based pagination
	result, err := s.crawlJobRepo.ListWithCursor(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list crawl jobs: %w", err)
	}

	return result, nil
}
