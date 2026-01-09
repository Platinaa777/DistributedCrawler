package crawljob

import (
	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
)

type CrawlJobImplementation struct {
	crawlergrpc.UnimplementedCrawlerServiceServer
	crawlJobService  service.CrawlJobService
	crawlTaskService service.CrawlTaskService
}

func NewImplementation(crawlJobService service.CrawlJobService, crawlTaskService service.CrawlTaskService) *CrawlJobImplementation {
	return &CrawlJobImplementation{
		crawlJobService:  crawlJobService,
		crawlTaskService: crawlTaskService,
	}
}
