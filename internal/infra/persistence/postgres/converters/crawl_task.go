package converters

import (
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"
)

func SaveCrawlTaskToSnapshot(crawlTask models.CrawlTask) *snapshots.CrawlTaskSnapshot {
	return &snapshots.CrawlTaskSnapshot{
		ID:         crawlTask.ID.String(),
		JobID:      crawlTask.JobID.String(),
		URL:        crawlTask.URL,
		Status:     crawlTask.Status.String(),
		EnqueuedAt: crawlTask.EnqueuedAt,
	}
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

	return &models.CrawlTask{
		ID:         id,
		JobID:      jobID,
		Job:        nil, // not populated for non-joined queries
		URL:        snapshot.URL,
		Status:     models.TaskStatus(snapshot.Status),
		EnqueuedAt: snapshot.EnqueuedAt,
	}, nil
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

	return &models.CrawlTask{
		ID:         id,
		JobID:      jobID,
		Job:        job,
		URL:        snapshot.URL,
		Status:     models.TaskStatus(snapshot.Status),
		EnqueuedAt: snapshot.EnqueuedAt,
	}, nil
}
