package crawljob

import (
	"distributed-crawler/internal/domain/crawl/models"
	crawlergrpc "distributed-crawler/pkg/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func ToProtoCrawlTask(task *models.CrawlTask) *crawlergrpc.CrawlTask {
	if task == nil {
		return nil
	}

	protoTask := &crawlergrpc.CrawlTask{
		Id:         task.ID.String(),
		JobId:      task.JobID.String(),
		Url:        task.URL,
		Status:     task.Status.String(),
		EnqueuedAt: timestamppb.New(task.EnqueuedAt),
	}

	if task.Job != nil {
		protoTask.Job = ToProtoCrawlJob(task.Job)
	}

	return protoTask
}
