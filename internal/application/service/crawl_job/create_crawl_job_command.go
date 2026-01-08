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
	status := models.TaskStatus(command.Status)
	if !status.IsValid() {
		return valueobjects.CrawlJobID{}, fmt.Errorf("invalid status: %s, must be one of: %s", command.Status, models.AllTaskStatusesString())
	}

	crawlJob := models.CrawlJob{
		ID:        valueobjects.GenerateCrawlJobID(),
		Name:      command.Name,
		Status:    status,
		CreatedAt: time.Now(),
	}

	id, err := s.crawlJobRepo.Create(ctx, crawlJob)
	if err != nil {
		return valueobjects.CrawlJobID{}, fmt.Errorf("failed to create crawl job: %w", err)
	}

	return id, nil
}
