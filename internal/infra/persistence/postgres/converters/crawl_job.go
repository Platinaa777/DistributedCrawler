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

	// Handle export fields
	if crawlJob.ExportJSONKey != nil {
		snapshot.ExportJSONKey = sql.NullString{
			String: *crawlJob.ExportJSONKey,
			Valid:  true,
		}
	}
	if crawlJob.ExportCSVKey != nil {
		snapshot.ExportCSVKey = sql.NullString{
			String: *crawlJob.ExportCSVKey,
			Valid:  true,
		}
	}
	if crawlJob.ExportedAt != nil {
		snapshot.ExportedAt = sql.NullTime{
			Time:  *crawlJob.ExportedAt,
			Valid: true,
		}
	}
	snapshot.ExportStatus = sql.NullString{
		String: crawlJob.ExportStatus.String(),
		Valid:  true,
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

	// Handle export fields
	if crawlJob.ExportJSONKey.Valid {
		job.ExportJSONKey = &crawlJob.ExportJSONKey.String
	}
	if crawlJob.ExportCSVKey.Valid {
		job.ExportCSVKey = &crawlJob.ExportCSVKey.String
	}
	if crawlJob.ExportedAt.Valid {
		job.ExportedAt = &crawlJob.ExportedAt.Time
	}
	if crawlJob.ExportStatus.Valid {
		job.ExportStatus = models.ExportStatus(crawlJob.ExportStatus.String)
	} else {
		// Default to NOT_STARTED if null
		job.ExportStatus = models.ExportStatusNotStarted
	}

	return job, nil
}
