package converters

import (
	"database/sql"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"
	"encoding/json"
	"fmt"
)

// SavePageFetchToSnapshot converts domain model to database snapshot
func SavePageFetchToSnapshot(fetch *models.PageFetch) (*snapshots.PageFetchSnapshot, error) {
	snapshot := &snapshots.PageFetchSnapshot{
		TaskID:         fetch.TaskID.String(),
		JobID:          fetch.JobID.String(),
		URL:            fetch.URL,
		StatusCode:     fetch.StatusCode,
		DurationMs:     fetch.DurationMs,
		BodyHash:       fetch.BodyHash,
		MinioObjectKey: fetch.MinioObjectKey,
		FetchedAt:      fetch.FetchedAt,
		CreatedAt:      fetch.CreatedAt,
	}

	// Handle nullable fields
	if fetch.FinalURL != nil {
		snapshot.FinalURL = sql.NullString{String: *fetch.FinalURL, Valid: true}
	}

	if fetch.ContentType != nil {
		snapshot.ContentType = sql.NullString{String: *fetch.ContentType, Valid: true}
	}

	if fetch.ContentLength != nil {
		snapshot.ContentLength = sql.NullInt64{Int64: *fetch.ContentLength, Valid: true}
	}

	// Convert headers map to JSONB
	if fetch.Headers != nil {
		headersJSON, err := json.Marshal(fetch.Headers)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal headers: %w", err)
		}
		snapshot.Headers = headersJSON
	}

	return snapshot, nil
}

// RestorePageFetchFromSnapshot converts database snapshot to domain model
func RestorePageFetchFromSnapshot(snapshot *snapshots.PageFetchSnapshot) (*models.PageFetch, error) {
	taskID, err := valueobjects.NewCrawlTaskID(snapshot.TaskID)
	if err != nil {
		return nil, fmt.Errorf("invalid task ID: %w", err)
	}

	jobID, err := valueobjects.NewCrawlJobID(snapshot.JobID)
	if err != nil {
		return nil, fmt.Errorf("invalid job ID: %w", err)
	}

	fetch := &models.PageFetch{
		TaskID:         taskID,
		JobID:          jobID,
		URL:            snapshot.URL,
		StatusCode:     snapshot.StatusCode,
		DurationMs:     snapshot.DurationMs,
		BodyHash:       snapshot.BodyHash,
		MinioObjectKey: snapshot.MinioObjectKey,
		FetchedAt:      snapshot.FetchedAt,
		CreatedAt:      snapshot.CreatedAt,
	}

	// Handle nullable fields
	if snapshot.FinalURL.Valid {
		fetch.FinalURL = &snapshot.FinalURL.String
	}

	if snapshot.ContentType.Valid {
		fetch.ContentType = &snapshot.ContentType.String
	}

	if snapshot.ContentLength.Valid {
		fetch.ContentLength = &snapshot.ContentLength.Int64
	}

	// Unmarshal headers JSONB
	if len(snapshot.Headers) > 0 {
		var headers map[string]string
		if err := json.Unmarshal(snapshot.Headers, &headers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
		}
		fetch.Headers = headers
	}

	return fetch, nil
}
