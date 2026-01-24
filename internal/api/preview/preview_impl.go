package preview

import (
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
)

type PreviewImplementation struct {
	crawlergrpc.UnimplementedPreviewServiceServer
	previewService service.PreviewService
}

func NewImplementation(previewService service.PreviewService) *PreviewImplementation {
	return &PreviewImplementation{
		previewService: previewService,
	}
}
