package converters

import (
	"database/sql"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"
)

func SavePreviewToSnapshot(preview models.Preview) *snapshots.PreviewSnapshot {
	snapshot := &snapshots.PreviewSnapshot{
		ID:          preview.ID.String(),
		SourceURL:   preview.SourceURL,
		MinioKey:    preview.MinioKey,
		ContentType: preview.ContentType,
		DownloadURL: preview.DownloadURL,
		CreatedAt:   preview.CreatedAt,
	}

	// Handle FinalURL
	if preview.FinalURL != nil {
		snapshot.FinalURL = sql.NullString{
			String: *preview.FinalURL,
			Valid:  true,
		}
	}

	// Handle ExpiresAt
	if preview.ExpiresAt != nil {
		snapshot.ExpiresAt = sql.NullTime{
			Time:  *preview.ExpiresAt,
			Valid: true,
		}
	}

	return snapshot
}

func RestorePreviewFromSnapshot(snapshot snapshots.PreviewSnapshot) (*models.Preview, error) {
	id, err := valueobjects.NewPreviewID(snapshot.ID)
	if err != nil {
		return nil, err
	}

	preview := &models.Preview{
		ID:          id,
		SourceURL:   snapshot.SourceURL,
		MinioKey:    snapshot.MinioKey,
		ContentType: snapshot.ContentType,
		DownloadURL: snapshot.DownloadURL,
		CreatedAt:   snapshot.CreatedAt,
	}

	// Handle FinalURL
	if snapshot.FinalURL.Valid {
		preview.FinalURL = &snapshot.FinalURL.String
	}

	// Handle ExpiresAt
	if snapshot.ExpiresAt.Valid {
		preview.ExpiresAt = &snapshot.ExpiresAt.Time
	}

	return preview, nil
}