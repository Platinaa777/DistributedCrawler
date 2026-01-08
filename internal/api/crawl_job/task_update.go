package crawljob

import (
	"context"
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
)

func (i *CrawlJobImplementation) UpdateTask(ctx context.Context, req *crawlergrpc.UpdateTaskRequest) (*crawlergrpc.UpdateTaskResponse, error) {
	command := service.UpdateTaskStatusCommand{
		TaskID: req.Id,
		Status: req.Status,
	}

	if err := i.crawlTaskService.UpdateTaskStatus(ctx, command); err != nil {
		return nil, err
	}

	// Get updated task to return
	getQuery := service.GetCrawlTaskQuery{
		ID: req.Id,
	}

	task, err := i.crawlTaskService.GetTask(ctx, getQuery)
	if err != nil {
		return nil, err
	}

	return &crawlergrpc.UpdateTaskResponse{
		Task: ToProtoCrawlTask(task),
	}, nil
}
