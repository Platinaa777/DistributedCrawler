package preview

import (
	"context"
	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"fmt"
)

func (s *previewServ) GetPreview(ctx context.Context, query service.GetPreviewQuery) (*models.Preview, error) {
	// Parse preview ID
	previewID, err := valueobjects.NewPreviewID(query.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid preview_id: %w", err)
	}

	// Get preview from repository (including presigned URL stored in DB)
	preview, err := s.previewRepo.Get(ctx, previewID)
	if err != nil {
		return nil, fmt.Errorf("failed to get preview: %w", err)
	}

	return preview, nil
}
