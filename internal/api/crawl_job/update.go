package crawljob

import (
	"context"
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
)

func (i *CrawlJobImplementation) UpdateJob(ctx context.Context, req *crawlergrpc.UpdateJobRequest) (*crawlergrpc.UpdateJobResponse, error) {
	command := service.UpdateCrawlJobStatusCommand{
		JobID:  req.Id,
		Status: req.Status,
	}

	if err := i.crawlJobService.UpdateJobStatus(ctx, command); err != nil {
		return nil, err
	}

	// Get updated job to return
	getQuery := service.GetCrawlJobQuery{
		ID: req.Id,
	}

	job, err := i.crawlJobService.GetCrawlJob(ctx, getQuery)
	if err != nil {
		return nil, err
	}

	return &crawlergrpc.UpdateJobResponse{
		Job: ToProtoCrawlJob(job),
	}, nil
}
