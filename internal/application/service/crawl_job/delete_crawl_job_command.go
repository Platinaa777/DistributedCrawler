package crawljob

import (
	"context"
	"fmt"

	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/valueobjects"
)

// DeleteCrawlJob deletes a crawl job and its config.
// For scheduled jobs the config may have multiple runs; deleting the config
// cascades to all of them (crawl_jobs → crawl_tasks via FK ON DELETE CASCADE).
func (s *crawlJobServ) DeleteCrawlJob(ctx context.Context, cmd service.DeleteCrawlJobCommand) error {
	id, err := valueobjects.NewCrawlJobID(cmd.JobID)
	if err != nil {
		return fmt.Errorf("invalid job id: %w", err)
	}

	job, err := s.crawlJobRepo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("get crawl job: %w", err)
	}

	// Deleting the config cascades to all associated crawl_jobs and their crawl_tasks.
	// For scheduled configs this removes every run; for one-time configs it removes
	// only the single run.
	return s.crawlJobConfigRepo.Delete(ctx, job.JobConfigID)
}
