package crawljob

import (
	"context"
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
)

func (i *CrawlJobImplementation) GetTask(ctx context.Context, req *crawlergrpc.GetTaskRequest) (*crawlergrpc.GetTaskResponse, error) {
	query := service.GetCrawlTaskQuery{
		ID: req.Id,
	}

	task, err := i.crawlTaskService.GetTask(ctx, query)
	if err != nil {
		return nil, err
	}

	return &crawlergrpc.GetTaskResponse{
		Task: ToProtoCrawlTask(task),
	}, nil
}
