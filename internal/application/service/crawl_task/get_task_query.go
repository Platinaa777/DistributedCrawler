package crawltask

import (
	"context"
	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"fmt"
)

func (s *crawlTaskServ) GetTask(ctx context.Context, query service.GetCrawlTaskQuery) (*models.CrawlTask, error) {
	taskID, err := valueobjects.NewCrawlTaskID(query.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid task ID: %w", err)
	}

	task, err := s.crawlTaskRepo.Get(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get crawl task: %w", err)
	}

	return task, nil
}
