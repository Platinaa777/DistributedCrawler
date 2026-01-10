package preview

import (
	"context"
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
)

func (i *PreviewImplementation) CreatePreview(ctx context.Context, req *crawlergrpc.CreatePreviewRequest) (*crawlergrpc.CreatePreviewResponse, error) {
	// Build command
	command := service.CreatePreviewCommand{
		URL: req.Url,
	}

	// Execute service command
	preview, err := i.previewService.CreatePreview(ctx, command)
	if err != nil {
		return nil, err
	}

	// Return only ID
	return &crawlergrpc.CreatePreviewResponse{
		Id: preview.ID.String(),
	}, nil
}
