package crawltask

import (
	"context"
	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"fmt"
	"time"
)

func (s *crawlTaskServ) CreateTask(ctx context.Context, command service.CreateCrawlTaskCommand) (valueobjects.CrawlTaskID, error) {
	jobID, err := valueobjects.NewCrawlJobID(command.JobID)
	if err != nil {
		return valueobjects.CrawlTaskID{}, fmt.Errorf("invalid job ID: %w", err)
	}

	task := models.CrawlTask{
		ID:         valueobjects.GenerateCrawlTaskID(),
		JobID:      jobID,
		URL:        command.URL,
		Status:     models.TaskStatusPending,
		EnqueuedAt: time.Now(),
	}

	taskID, err := s.crawlTaskRepo.Create(ctx, task)
	if err != nil {
		return valueobjects.CrawlTaskID{}, fmt.Errorf("failed to create crawl task: %w", err)
	}

	return taskID, nil
}
