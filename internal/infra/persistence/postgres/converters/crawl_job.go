package converters

import (
	"database/sql"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"
)

func SaveCrawlJobToSnapshot(crawlJob models.CrawlJob) *snapshots.CrawlJobSnapshot {
	snapshot := &snapshots.CrawlJobSnapshot{
		ID:        crawlJob.ID.String(),
		Name:      crawlJob.Name,
		Status:    crawlJob.Status,
		CreatedAt: crawlJob.CreatedAt,
	}

	if crawlJob.CompletedAt != nil {
		snapshot.CompletedAt = sql.NullTime{
			Time:  *crawlJob.CompletedAt,
			Valid: true,
		}
	}

	return snapshot
}

func RestoreCrawlJobFromSnapshot(crawlJob snapshots.CrawlJobSnapshot) (*models.CrawlJob, error) {
	id, err := valueobjects.NewCrawlJobID(crawlJob.ID)
	if err != nil {
		return nil, err
	}

	job := &models.CrawlJob{
		ID:        id,
		Name:      crawlJob.Name,
		Status:    crawlJob.Status,
		CreatedAt: crawlJob.CreatedAt,
	}

	if crawlJob.CompletedAt.Valid {
		job.CompletedAt = &crawlJob.CompletedAt.Time
	}

	return job, nil
}