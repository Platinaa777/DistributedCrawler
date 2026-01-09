package crawljob

import (
	"context"
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
	"log"
)

func (i *CrawlJobImplementation) CreateJob(ctx context.Context, req *crawlergrpc.CreateJobRequest) (*crawlergrpc.CreateJobResponse, error) {
	// Convert proto config to domain config
	config := FromProtoCrawlJobConfig(req.Config)

	command := service.CreateCrawlJobCommand{
		Config: config,
	}

	id, err := i.crawlJobService.CreateCrawlJob(ctx, command)
	if err != nil {
		return nil, err
	}

	log.Printf("inserted crawl job with id: %s and %d tasks", id.String(), len(config.Seeds))

	return &crawlergrpc.CreateJobResponse{
		Id: id.String(),
	}, nil
}
