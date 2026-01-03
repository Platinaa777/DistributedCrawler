package crawljob

import (
	"distributed-crawler/internal/domain/crawl/models"
	crawlergrpc "distributed-crawler/pkg/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// ToProtoCrawlJob converts domain CrawlJob to protobuf CrawlJob
func ToProtoCrawlJob(job *models.CrawlJob) *crawlergrpc.CrawlJob {
	if job == nil {
		return nil
	}

	protoJob := &crawlergrpc.CrawlJob{
		Id:        job.ID.String(),
		Name:      job.Name,
		Status:    job.Status,
		CreatedAt: timestamppb.New(job.CreatedAt),
	}

	if job.CompletedAt != nil {
		protoJob.CompletedAt = timestamppb.New(*job.CompletedAt)
	}

	return protoJob
}
