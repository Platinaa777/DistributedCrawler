package crawltask

import (
	"context"
	"fmt"

	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
)

func (s *crawlTaskServ) ListTasksByJob(ctx context.Context, query service.ListTasksByJobQuery) (*service.ListTasksResult, error) {
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

	// Validate depth range
	if query.Filter.MinDepth != nil && query.Filter.MaxDepth != nil {
		if *query.Filter.MinDepth > *query.Filter.MaxDepth {
			return nil, fmt.Errorf("min_depth cannot be greater than max_depth")
		}
	}

	// Validate date range
	if query.Filter.EnqueuedFrom != nil && query.Filter.EnqueuedTo != nil {
		if query.Filter.EnqueuedFrom.After(*query.Filter.EnqueuedTo) {
			return nil, fmt.Errorf("enqueued_from cannot be after enqueued_to")
		}
	}

	// Call repository with cursor-based pagination
	result, err := s.crawlTaskRepo.ListWithCursor(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	return result, nil
}
