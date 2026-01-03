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
	crawlJob := models.CrawlJob{
		ID:        valueobjects.GenerateCrawlJobID(),
		Name:      command.Name,
		Status:    command.Status,
		CreatedAt: time.Now(),
	}

	id, err := s.crawlJobRepo.Create(ctx, crawlJob)
	if err != nil {
		fmt.Printf("some error during creating crawl job: %v\n", err)
	}

	return id, err
}
