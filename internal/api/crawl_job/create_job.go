package crawljob

import (
	"context"
	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/auth"
	crawlergrpc "distributed-crawler/pkg/v1"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i *CrawlJobImplementation) CreateJob(ctx context.Context, req *crawlergrpc.CreateJobRequest) (*crawlergrpc.CreateJobResponse, error) {
	userID, ok := auth.GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		return nil, status.Error(codes.Unauthenticated, "missing user id in context")
	}

	// Convert proto config to domain config
	config := FromProtoCrawlJobConfig(req.Config)

	command := service.CreateCrawlJobCommand{
		Config: config,
		UserID: userID,
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
