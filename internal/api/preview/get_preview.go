package preview

import (
	"context"
	"distributed-crawler/internal/api/preview/converters"
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
)

func (i *PreviewImplementation) GetPreview(ctx context.Context, req *crawlergrpc.GetPreviewRequest) (*crawlergrpc.GetPreviewResponse, error) {
	// Build query
	query := service.GetPreviewQuery{
		ID: req.Id,
	}

	// Execute service query
	preview, err := i.previewService.GetPreview(ctx, query)
	if err != nil {
		return nil, err
	}

	// Convert to proto
	return &crawlergrpc.GetPreviewResponse{
		Preview: converters.ToProtoPreview(preview),
	}, nil
}
