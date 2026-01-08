package crawltask

import (
	"context"
	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"fmt"
)

func (s *crawlTaskServ) ListTasksByJob(ctx context.Context, query service.ListTasksByJobQuery) ([]*models.CrawlTask, error) {
	jobID, err := valueobjects.NewCrawlJobID(query.JobID)
	if err != nil {
		return nil, fmt.Errorf("invalid job ID: %w", err)
	}

	tasks, err := s.crawlTaskRepo.ListByJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks by job: %w", err)
	}

	return tasks, nil
}
