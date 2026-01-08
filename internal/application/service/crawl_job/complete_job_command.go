package crawljob

import (
	"context"
	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"fmt"
	"time"
)

func (s *crawlJobServ) CompleteJob(ctx context.Context, command service.CompleteCrawlJobCommand) error {
	jobID, err := valueobjects.NewCrawlJobID(command.JobID)
	if err != nil {
		return fmt.Errorf("invalid job ID: %w", err)
	}

	job, err := s.crawlJobRepo.Get(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get crawl job: %w", err)
	}

	now := time.Now()
	job.Status = models.TaskStatusCompleted
	job.CompletedAt = &now

	if err := s.crawlJobRepo.Update(ctx, *job); err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	return nil
}
