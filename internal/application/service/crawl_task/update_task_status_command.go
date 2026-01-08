package crawltask

import (
	"context"
	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"fmt"
)

func (s *crawlTaskServ) UpdateTaskStatus(ctx context.Context, command service.UpdateTaskStatusCommand) error {
	status := models.TaskStatus(command.Status)
	if !status.IsValid() {
		return fmt.Errorf("invalid status: %s, must be one of: %s", command.Status, models.AllTaskStatusesString())
	}

	taskID, err := valueobjects.NewCrawlTaskID(command.TaskID)
	if err != nil {
		return fmt.Errorf("invalid task ID: %w", err)
	}

	task, err := s.crawlTaskRepo.Get(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get crawl task: %w", err)
	}

	task.Status = status

	if err := s.crawlTaskRepo.Update(ctx, *task); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	return nil
}
