package crawljob

import (
	"context"
	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
	"fmt"
)

func (s *crawlJobServ) ListCrawlJobs(ctx context.Context, query service.ListCrawlJobsQuery) ([]*models.CrawlJob, error) {
	if query.Limit == 0 {
		query.Limit = 50
	}

	var jobs []*models.CrawlJob
	var err error

	if query.Status != "" {
		status := models.TaskStatus(query.Status)
		if !status.IsValid() {
			return nil, fmt.Errorf("invalid status: %s", query.Status)
		}
		jobs, err = s.crawlJobRepo.List(ctx, status, query.Limit, query.Offset)
	} else {
		jobs, err = s.crawlJobRepo.ListAll(ctx, query.Limit, query.Offset)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list crawl jobs: %w", err)
	}

	return jobs, nil
}
