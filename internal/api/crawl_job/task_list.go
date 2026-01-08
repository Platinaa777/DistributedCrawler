package crawljob

import (
	"context"
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
)

func (i *CrawlJobImplementation) ListTasksByJob(ctx context.Context, req *crawlergrpc.ListTasksByJobRequest) (*crawlergrpc.ListTasksByJobResponse, error) {
	query := service.ListTasksByJobQuery{
		JobID: req.JobId,
	}

	tasks, err := i.crawlTaskService.ListTasksByJob(ctx, query)
	if err != nil {
		return nil, err
	}

	protoTasks := make([]*crawlergrpc.CrawlTask, 0, len(tasks))
	for _, task := range tasks {
		protoTasks = append(protoTasks, ToProtoCrawlTask(task))
	}

	return &crawlergrpc.ListTasksByJobResponse{
		Tasks: protoTasks,
	}, nil
}
