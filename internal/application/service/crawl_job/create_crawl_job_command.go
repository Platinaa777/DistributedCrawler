package crawljob

import (
	"context"
	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"fmt"
	"time"
)

func (s *crawlJobServ) CreateCrawlJob(ctx context.Context, command service.CreateCrawlJobCommand) (valueobjects.CrawlJobID, error) {
	if len(command.URLs) == 0 {
		return valueobjects.CrawlJobID{}, fmt.Errorf("URLs list cannot be empty")
	}

	var jobID valueobjects.CrawlJobID

	err := s.txManager.ReadCommitted(ctx, func(ctx context.Context) error {
		// Create crawl job
		crawlJob := models.CrawlJob{
			ID:        valueobjects.GenerateCrawlJobID(),
			Name:      command.Name,
			Status:    models.TaskStatusPending,
			CreatedAt: time.Now(),
		}

		id, err := s.crawlJobRepo.Create(ctx, crawlJob)
		if err != nil {
			return fmt.Errorf("failed to create crawl job: %w", err)
		}
		jobID = id

		// Create crawl tasks
		tasks := make([]models.CrawlTask, 0, len(command.URLs))
		now := time.Now()

		for _, url := range command.URLs {
			task := models.CrawlTask{
				ID:         valueobjects.GenerateCrawlTaskID(),
				JobID:      jobID,
				URL:        url,
				Status:     models.TaskStatusPending,
				EnqueuedAt: now,
			}
			tasks = append(tasks, task)
		}

		if err := s.crawlTaskRepo.BulkCreate(ctx, tasks); err != nil {
			return fmt.Errorf("failed to create crawl tasks: %w", err)
		}

		return nil
	})

	if err != nil {
		return valueobjects.CrawlJobID{}, err
	}

	return jobID, nil
}
