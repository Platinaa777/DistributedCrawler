package crawltask

import (
	"context"
	"fmt"

	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/valueobjects"
)

func (s *crawlTaskServ) GetTaskAnalytics(ctx context.Context, query service.GetTaskAnalyticsQuery) (*service.TaskAnalytics, error) {
	jobID, err := valueobjects.NewCrawlJobID(query.JobID)
	if err != nil {
		return nil, fmt.Errorf("invalid job ID: %w", err)
	}

	analytics, err := s.crawlTaskRepo.GetAnalyticsByJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task analytics: %w", err)
	}

	return analytics, nil
}
