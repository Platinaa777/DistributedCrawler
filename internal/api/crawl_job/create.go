package crawljob

import (
	"context"
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
	"log"
)

func (i *CrawlJobImplementation) CreateJob(ctx context.Context, req *crawlergrpc.CreateJobRequest) (*crawlergrpc.CreateJobResponse, error) {
	command := service.CreateCrawlJobCommand{
		Name:   req.Name,
		Status: req.Status,
	}

	id, err := i.crawlJobService.CreateCrawlJob(ctx, command)
	if err != nil {
		return nil, err
	}

	log.Printf("inserted crawl job with id: %s", id.String())

	return &crawlergrpc.CreateJobResponse{
		Id: id.String(),
	}, nil
}