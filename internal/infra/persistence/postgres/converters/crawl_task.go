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
		BodyHash:       crawlTask.BodyHash,
		MinioObjectKey: crawlTask.MinioObjectKey,
	}

	// Handle FinalURL
	if crawlTask.FinalURL != nil {
		snapshot.FinalURL = sql.NullString{
			String: *crawlTask.FinalURL,
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
		BodyHash:       snapshot.BodyHash,
		MinioObjectKey: snapshot.MinioObjectKey,
	}

	// Handle FinalURL
	if snapshot.FinalURL.Valid {
		task.FinalURL = &snapshot.FinalURL.String
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
		BodyHash:       snapshot.BodyHash,
		MinioObjectKey: snapshot.MinioObjectKey,
	}

	// Handle FinalURL
	if snapshot.FinalURL.Valid {
		task.FinalURL = &snapshot.FinalURL.String
	}

	return task, nil
}
