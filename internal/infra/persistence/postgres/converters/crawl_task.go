package converters

import (
	"database/sql"

	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"
)

func SaveCrawlTaskToSnapshot(crawlTask models.CrawlTask) *snapshots.CrawlTaskSnapshot {
	snapshot := &snapshots.CrawlTaskSnapshot{
		ID:             crawlTask.ID.String(),
		JobID:          crawlTask.JobID.String(),
		URL:            crawlTask.URL,
		Status:         crawlTask.Status.String(),
		EnqueuedAt:     crawlTask.EnqueuedAt,
		Depth:          crawlTask.Depth,
		MinioObjectKey: crawlTask.MinioObjectKey,
	}

	// Handle FinalURL
	if crawlTask.FinalURL != nil {
		snapshot.FinalURL = sql.NullString{
			String: *crawlTask.FinalURL,
			Valid:  true,
		}
	}

	// Handle result fields
	if crawlTask.ResultObjectKey != nil {
		snapshot.ResultObjectKey = sql.NullString{
			String: *crawlTask.ResultObjectKey,
			Valid:  true,
		}
	}
	if crawlTask.ResultContentType != nil {
		snapshot.ResultContentType = sql.NullString{
			String: *crawlTask.ResultContentType,
			Valid:  true,
		}
	}
	if crawlTask.ResultSizeBytes != nil {
		snapshot.ResultSizeBytes = sql.NullInt64{
			Int64: *crawlTask.ResultSizeBytes,
			Valid: true,
		}
	}
	if crawlTask.ResultCreatedAt != nil {
		snapshot.ResultCreatedAt = sql.NullTime{
			Time:  *crawlTask.ResultCreatedAt,
			Valid: true,
		}
	}

	// Handle ErrorMessage
	if crawlTask.ErrorMessage != nil {
		snapshot.ErrorMessage = sql.NullString{
			String: *crawlTask.ErrorMessage,
			Valid:  true,
		}
	}

	return snapshot
}

func RestoreCrawlTaskFromSnapshot(snapshot snapshots.CrawlTaskSnapshot) (*models.CrawlTask, error) {
	id, err := valueobjects.NewCrawlTaskID(snapshot.ID)
	if err != nil {
		return nil, err
	}

	jobID, err := valueobjects.NewCrawlJobID(snapshot.JobID)
	if err != nil {
		return nil, err
	}

	task := &models.CrawlTask{
		ID:             id,
		JobID:          jobID,
		Job:            nil, // not populated for non-joined queries
		URL:            snapshot.URL,
		Status:         models.TaskStatus(snapshot.Status),
		EnqueuedAt:     snapshot.EnqueuedAt,
		Depth:          snapshot.Depth,
		MinioObjectKey: snapshot.MinioObjectKey,
	}

	// Handle FinalURL
	if snapshot.FinalURL.Valid {
		task.FinalURL = &snapshot.FinalURL.String
	}

	// Handle result fields
	if snapshot.ResultObjectKey.Valid {
		task.ResultObjectKey = &snapshot.ResultObjectKey.String
	}
	if snapshot.ResultContentType.Valid {
		task.ResultContentType = &snapshot.ResultContentType.String
	}
	if snapshot.ResultSizeBytes.Valid {
		task.ResultSizeBytes = &snapshot.ResultSizeBytes.Int64
	}
	if snapshot.ResultCreatedAt.Valid {
		task.ResultCreatedAt = &snapshot.ResultCreatedAt.Time
	}

	// Handle ErrorMessage
	if snapshot.ErrorMessage.Valid {
		task.ErrorMessage = &snapshot.ErrorMessage.String
	}

	return task, nil
}

func RestoreCrawlTaskWithJobFromSnapshot(snapshot snapshots.CrawlTaskWithJobSnapshot) (*models.CrawlTask, error) {
	id, err := valueobjects.NewCrawlTaskID(snapshot.ID)
	if err != nil {
		return nil, err
	}

	jobID, err := valueobjects.NewCrawlJobID(snapshot.JobID)
	if err != nil {
		return nil, err
	}

	var job *models.CrawlJob
	if snapshot.Job != nil {
		job, err = RestoreCrawlJobFromSnapshot(*snapshot.Job)
		if err != nil {
			return nil, err
		}
	}

	task := &models.CrawlTask{
		ID:             id,
		JobID:          jobID,
		Job:            job,
		URL:            snapshot.URL,
		Status:         models.TaskStatus(snapshot.Status),
		EnqueuedAt:     snapshot.EnqueuedAt,
		Depth:          snapshot.Depth,
		MinioObjectKey: snapshot.MinioObjectKey,
	}

	// Handle FinalURL
	if snapshot.FinalURL.Valid {
		task.FinalURL = &snapshot.FinalURL.String
	}

	// Handle result fields
	if snapshot.ResultObjectKey.Valid {
		task.ResultObjectKey = &snapshot.ResultObjectKey.String
	}
	if snapshot.ResultContentType.Valid {
		task.ResultContentType = &snapshot.ResultContentType.String
	}
	if snapshot.ResultSizeBytes.Valid {
		task.ResultSizeBytes = &snapshot.ResultSizeBytes.Int64
	}
	if snapshot.ResultCreatedAt.Valid {
		task.ResultCreatedAt = &snapshot.ResultCreatedAt.Time
	}

	// Handle ErrorMessage
	if snapshot.ErrorMessage.Valid {
		task.ErrorMessage = &snapshot.ErrorMessage.String
	}

	return task, nil
}
