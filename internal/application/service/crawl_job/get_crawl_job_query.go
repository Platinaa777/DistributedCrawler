package crawljob

import (
	"context"
	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
)

func (s *crawlJobServ) GetCrawlJob(ctx context.Context, query service.GetCrawlJobQuery) (*models.CrawlJob, error) {
	id, err := valueobjects.NewCrawlJobID(query.ID)
	if err != nil {
		return nil, err
	}

	crawlJob, err := s.crawlJobRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return crawlJob, nil
}
