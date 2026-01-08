package crawljob

import (
	"context"
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
)

func (i *CrawlJobImplementation) ListJobs(ctx context.Context, req *crawlergrpc.ListJobsRequest) (*crawlergrpc.ListJobsResponse, error) {
	query := service.ListCrawlJobsQuery{
		Limit:  int(req.Limit),
		Offset: int(req.Offset),
	}

	if req.Status != nil {
		query.Status = *req.Status
	}

	jobs, err := i.crawlJobService.ListCrawlJobs(ctx, query)
	if err != nil {
		return nil, err
	}

	protoJobs := make([]*crawlergrpc.CrawlJob, 0, len(jobs))
	for _, job := range jobs {
		protoJobs = append(protoJobs, ToProtoCrawlJob(job))
	}

	return &crawlergrpc.ListJobsResponse{
		Jobs: protoJobs,
	}, nil
}
