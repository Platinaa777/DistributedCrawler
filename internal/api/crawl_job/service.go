package crawljob

import (
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
)

type CrawlJobImplementation struct {
	crawlergrpc.UnimplementedCrawlerServiceServer
	crawlJobService service.CrawlJobService
}

func NewImplementation(crawlJobService service.CrawlJobService) *CrawlJobImplementation {
	return &CrawlJobImplementation{
		crawlJobService: crawlJobService,
	}
}
