package crawljob

import (
	"context"

	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
)

func (i *CrawlJobImplementation) GetTaskAnalytics(ctx context.Context, req *crawlergrpc.GetTaskAnalyticsRequest) (*crawlergrpc.GetTaskAnalyticsResponse, error) {
	query := service.GetTaskAnalyticsQuery{
		JobID: req.JobId,
	}

	analytics, err := i.crawlTaskService.GetTaskAnalytics(ctx, query)
	if err != nil {
		return nil, err
	}

	// Convert depth counts (uint64 -> uint64 in proto map)
	depthCounts := make(map[uint64]int64)
	for depth, count := range analytics.DepthCounts {
		depthCounts[depth] = count
	}

	return &crawlergrpc.GetTaskAnalyticsResponse{
		Analytics: &crawlergrpc.TaskAnalytics{
			StatusCounts: analytics.StatusCounts,
			DepthCounts:  depthCounts,
			TotalCount:   analytics.TotalCount,
		},
	}, nil
}
