package crawltask

import (
	crawltaskrepo "distributed-crawler/internal/domain/crawl/repos/crawl_task"
)

type crawlTaskServ struct {
	crawlTaskRepo crawltaskrepo.CrawlTaskRepository
}

func NewCrawlTaskService(crawlTaskRepo crawltaskrepo.CrawlTaskRepository) *crawlTaskServ {
	return &crawlTaskServ{
		crawlTaskRepo: crawlTaskRepo,
	}
}
