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
		Status:    crawlJob.Status.String(),
		CreatedAt: crawlJob.CreatedAt,
	}

	// Handle JobConfigID
	if !crawlJob.JobConfigID.IsEmpty() {
		snapshot.JobConfigID = sql.NullString{
			String: crawlJob.JobConfigID.String(),
			Valid:  true,
		}
	}

	// Handle CompletedAt
	if crawlJob.CompletedAt != nil {
		snapshot.CompletedAt = sql.NullTime{
			Time:  *crawlJob.CompletedAt,
			Valid: true,
		}
	}

	// Handle Error
	if crawlJob.Error != nil {
		snapshot.Error = crawlJob.Error
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
		Status:    models.TaskStatus(crawlJob.Status),
		CreatedAt: crawlJob.CreatedAt,
	}

	// Handle JobConfigID
	if crawlJob.JobConfigID.Valid {
		configID, err := valueobjects.NewID(crawlJob.JobConfigID.String)
		if err != nil {
			return nil, err
		}
		job.JobConfigID = configID
	}

	// Handle CompletedAt
	if crawlJob.CompletedAt.Valid {
		job.CompletedAt = &crawlJob.CompletedAt.Time
	}

	// Handle Error
	if crawlJob.Error != nil {
		job.Error = crawlJob.Error
	}

	return job, nil
}