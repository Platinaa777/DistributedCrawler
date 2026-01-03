package crawljob

import (
	"context"
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
	"log"
)

func (i *CrawlJobImplementation) GetJob(ctx context.Context, req *crawlergrpc.GetJobRequest) (*crawlergrpc.GetJobResponse, error) {
	query := service.GetCrawlJobQuery{
		ID: req.Id,
	}

	crawlJob, err := i.crawlJobService.GetCrawlJob(ctx, query)
	if err != nil {
		return nil, err
	}

	log.Printf("got crawl job %v", crawlJob)

	return &crawlergrpc.GetJobResponse{
		Job: ToProtoCrawlJob(crawlJob),
	}, nil
}