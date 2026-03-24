package crawljob

import (
	"context"
	"database/sql"
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
	"errors"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i *CrawlJobImplementation) GetJob(ctx context.Context, req *crawlergrpc.GetJobRequest) (*crawlergrpc.GetJobResponse, error) {
	query := service.GetCrawlJobQuery{
		ID: req.Id,
	}

	crawlJob, err := i.crawlJobService.GetCrawlJob(ctx, query)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "job not found")
		}
		return nil, err
	}

	log.Printf("got crawl job %v", crawlJob)

	return &crawlergrpc.GetJobResponse{
		Job: ToProtoCrawlJob(crawlJob),
	}, nil
}
