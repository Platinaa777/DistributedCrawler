package crawljob

import (
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
)

// PresignedURLGenerator generates presigned URLs for MinIO objects
type PresignedURLGenerator interface {
	PresignGetURL(key string, ttlMinutes int) (string, error)
}

type CrawlJobImplementation struct {
	crawlergrpc.UnimplementedCrawlerServiceServer
	crawlJobService  service.CrawlJobService
	crawlTaskService service.CrawlTaskService
	urlGenerator     PresignedURLGenerator
}

func NewImplementation(
	crawlJobService service.CrawlJobService,
	crawlTaskService service.CrawlTaskService,
	urlGenerator PresignedURLGenerator,
) *CrawlJobImplementation {
	return &CrawlJobImplementation{
		crawlJobService:  crawlJobService,
		crawlTaskService: crawlTaskService,
		urlGenerator:     urlGenerator,
	}
}
