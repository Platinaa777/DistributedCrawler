package converters

import (
	"database/sql"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"
	"encoding/json"
	"fmt"
)

// SavePageExtractToSnapshot converts domain model to database snapshot
func SavePageExtractToSnapshot(extract *models.PageExtract) (*snapshots.PageExtractSnapshot, error) {
	snapshot := &snapshots.PageExtractSnapshot{
		TaskID:            extract.TaskID.String(),
		LinkCount:         extract.LinkCount,
		ImageCount:        extract.ImageCount,
		ExternalLinkCount: extract.ExternalLinkCount,
		WordCount:         extract.WordCount,
		ParsedAt:          extract.ParsedAt,
		CreatedAt:         extract.CreatedAt,
	}

	// Handle nullable fields
	if extract.Title != nil {
		snapshot.Title = sql.NullString{String: *extract.Title, Valid: true}
	}

	if extract.MetaDescription != nil {
		snapshot.MetaDescription = sql.NullString{String: *extract.MetaDescription, Valid: true}
	}

	if extract.CanonicalURL != nil {
		snapshot.CanonicalURL = sql.NullString{String: *extract.CanonicalURL, Valid: true}
	}

	// Convert metadata map to JSONB
	if extract.Metadata != nil {
		metadataJSON, err := json.Marshal(extract.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		snapshot.Metadata = metadataJSON
	}

	return snapshot, nil
}

// RestorePageExtractFromSnapshot converts database snapshot to domain model
func RestorePageExtractFromSnapshot(snapshot *snapshots.PageExtractSnapshot) (*models.PageExtract, error) {
	taskID, err := valueobjects.NewCrawlTaskID(snapshot.TaskID)
	if err != nil {
		return nil, fmt.Errorf("invalid task ID: %w", err)
	}

	extract := &models.PageExtract{
		TaskID:            taskID,
		LinkCount:         snapshot.LinkCount,
		ImageCount:        snapshot.ImageCount,
		ExternalLinkCount: snapshot.ExternalLinkCount,
		WordCount:         snapshot.WordCount,
		ParsedAt:          snapshot.ParsedAt,
		CreatedAt:         snapshot.CreatedAt,
	}

	// Handle nullable fields
	if snapshot.Title.Valid {
		extract.Title = &snapshot.Title.String
	}

	if snapshot.MetaDescription.Valid {
		extract.MetaDescription = &snapshot.MetaDescription.String
	}

	if snapshot.CanonicalURL.Valid {
		extract.CanonicalURL = &snapshot.CanonicalURL.String
	}

	// Unmarshal metadata JSONB
	if len(snapshot.Metadata) > 0 {
		var metadata map[string]any
		if err := json.Unmarshal(snapshot.Metadata, &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
		extract.Metadata = metadata
	}

	return extract, nil
}

// SavePageLinkToSnapshot converts domain model to database snapshot
func SavePageLinkToSnapshot(link *models.PageLink) *snapshots.PageLinkSnapshot {
	snapshot := &snapshots.PageLinkSnapshot{
		ID:         link.ID.String(),
		TaskID:     link.TaskID.String(),
		URL:        link.URL,
		IsExternal: link.IsExternal,
		CreatedAt:  link.CreatedAt,
	}

	if link.AnchorText != nil {
		snapshot.AnchorText = sql.NullString{String: *link.AnchorText, Valid: true}
	}

	return snapshot
}

// RestorePageLinkFromSnapshot converts database snapshot to domain model
func RestorePageLinkFromSnapshot(snapshot *snapshots.PageLinkSnapshot) (*models.PageLink, error) {
	id, err := valueobjects.NewPageLinkID(snapshot.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid link ID: %w", err)
	}

	taskID, err := valueobjects.NewCrawlTaskID(snapshot.TaskID)
	if err != nil {
		return nil, fmt.Errorf("invalid task ID: %w", err)
	}

	link := &models.PageLink{
		ID:         id,
		TaskID:     taskID,
		URL:        snapshot.URL,
		IsExternal: snapshot.IsExternal,
		CreatedAt:  snapshot.CreatedAt,
	}

	if snapshot.AnchorText.Valid {
		link.AnchorText = &snapshot.AnchorText.String
	}

	return link, nil
}

// SavePageImageToSnapshot converts domain model to database snapshot
func SavePageImageToSnapshot(image *models.PageImage) *snapshots.PageImageSnapshot {
	snapshot := &snapshots.PageImageSnapshot{
		ID:        image.ID.String(),
		TaskID:    image.TaskID.String(),
		URL:       image.URL,
		CreatedAt: image.CreatedAt,
	}

	if image.AltText != nil {
		snapshot.AltText = sql.NullString{String: *image.AltText, Valid: true}
	}

	return snapshot
}

// RestorePageImageFromSnapshot converts database snapshot to domain model
func RestorePageImageFromSnapshot(snapshot *snapshots.PageImageSnapshot) (*models.PageImage, error) {
	id, err := valueobjects.NewPageImageID(snapshot.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid image ID: %w", err)
	}

	taskID, err := valueobjects.NewCrawlTaskID(snapshot.TaskID)
	if err != nil {
		return nil, fmt.Errorf("invalid task ID: %w", err)
	}

	image := &models.PageImage{
		ID:        id,
		TaskID:    taskID,
		URL:       snapshot.URL,
		CreatedAt: snapshot.CreatedAt,
	}

	if snapshot.AltText.Valid {
		image.AltText = &snapshot.AltText.String
	}

	return image, nil
}
