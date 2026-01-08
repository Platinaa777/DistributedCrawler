package crawljob

import (
	"distributed-crawler/internal/application/service"
	crawljobrepo "distributed-crawler/internal/domain/crawl/repos/crawl_job"
	crawltaskrepo "distributed-crawler/internal/domain/crawl/repos/crawl_task"
	"distributed-crawler/internal/infra/persistence"
)

type crawlJobServ struct {
	crawlJobRepo  crawljobrepo.CrawlJobRepository
	crawlTaskRepo crawltaskrepo.CrawlTaskRepository
	txManager     persistence.TxManager
}

func NewService(
	crawlJobRepo crawljobrepo.CrawlJobRepository,
	crawlTaskRepo crawltaskrepo.CrawlTaskRepository,
	txManager persistence.TxManager,
) service.CrawlJobService {
	return &crawlJobServ{
		crawlJobRepo:  crawlJobRepo,
		crawlTaskRepo: crawlTaskRepo,
		txManager:     txManager,
	}
}
