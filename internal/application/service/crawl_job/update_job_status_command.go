package crawljob

import (
	"context"
	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"fmt"
)

func (s *crawlJobServ) UpdateJobStatus(ctx context.Context, command service.UpdateCrawlJobStatusCommand) error {
	status := models.TaskStatus(command.Status)
	if !status.IsValid() {
		return fmt.Errorf("invalid status: %s, must be one of: %s", command.Status, models.AllTaskStatusesString())
	}

	jobID, err := valueobjects.NewCrawlJobID(command.JobID)
	if err != nil {
		return fmt.Errorf("invalid job ID: %w", err)
	}

	job, err := s.crawlJobRepo.Get(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get crawl job: %w", err)
	}

	job.Status = status

	if err := s.crawlJobRepo.Update(ctx, *job); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}
