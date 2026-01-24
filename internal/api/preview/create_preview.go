package preview

import (
	"context"
	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
	crawlergrpc "distributed-crawler/pkg/v1"
	"strings"

	"google.golang.org/grpc/metadata"
)

func (i *PreviewImplementation) CreatePreview(ctx context.Context, req *crawlergrpc.CreatePreviewRequest) (*crawlergrpc.CreatePreviewResponse, error) {
	authOptions := models.AuthOptions{}
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get("x-preview-cookie"); len(values) > 0 {
			authOptions.Cookie = strings.TrimSpace(values[0])
		}
	}

	// Build command
	command := service.CreatePreviewCommand{
		URL:  req.Url,
		Auth: authOptions,
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
