package preview

import (
	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/repos/preview"
	"distributed-crawler/internal/domain/crawl/services"
	"distributed-crawler/internal/infra/persistence"
)

type previewServ struct {
	previewRepo  preview.PreviewRepository
	fetcher      services.Fetcher
	contentStore services.ContentStore
	sanitizer    HTMLSanitizer
	urlGenerator PresignedURLGenerator
	txManager    persistence.TxManager
}

// HTMLSanitizer sanitizes HTML for safe iframe rendering
type HTMLSanitizer interface {
	Sanitize(html []byte) []byte
}

// PresignedURLGenerator generates presigned URLs for MinIO
type PresignedURLGenerator interface {
	PresignGetURL(key string, ttlMinutes int) (string, error)
}

func NewService(
	previewRepo preview.PreviewRepository,
	fetcher services.Fetcher,
	contentStore services.ContentStore,
	sanitizer HTMLSanitizer,
	urlGenerator PresignedURLGenerator,
	txManager persistence.TxManager,
) service.PreviewService {
	return &previewServ{
		previewRepo:  previewRepo,
		fetcher:      fetcher,
		contentStore: contentStore,
		sanitizer:    sanitizer,
		urlGenerator: urlGenerator,
		txManager:    txManager,
	}
}
