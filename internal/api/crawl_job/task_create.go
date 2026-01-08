package crawljob

import (
	"context"
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
	"log"
)

func (i *CrawlJobImplementation) CreateTask(ctx context.Context, req *crawlergrpc.CreateTaskRequest) (*crawlergrpc.CreateTaskResponse, error) {
	command := service.CreateCrawlTaskCommand{
		JobID: req.JobId,
		URL:   req.Url,
	}

	taskID, err := i.crawlTaskService.CreateTask(ctx, command)
	if err != nil {
		return nil, err
	}

	log.Printf("created crawl task with id: %s for job: %s", taskID.String(), req.JobId)

	// Get created task to return full object
	getQuery := service.GetCrawlTaskQuery{
		ID: taskID.String(),
	}

	task, err := i.crawlTaskService.GetTask(ctx, getQuery)
	if err != nil {
		return nil, err
	}

	return &crawlergrpc.CreateTaskResponse{
		Task: ToProtoCrawlTask(task),
	}, nil
}
