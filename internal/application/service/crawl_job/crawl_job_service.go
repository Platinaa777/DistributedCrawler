package crawljob

import (
	"distributed-crawler/internal/application/service"
	crawljobrepo "distributed-crawler/internal/domain/crawl/repos/crawl_job"
	"distributed-crawler/internal/infra/persistence"
)

type crawlJobServ struct {
	crawlJobRepo crawljobrepo.CrawlJobRepository
	txManager    persistence.TxManager
}

func NewService(
	crawlJobRepo crawljobrepo.CrawlJobRepository,
	txManager persistence.TxManager,
) service.CrawlJobService {
	return &crawlJobServ{
		crawlJobRepo: crawlJobRepo,
		txManager:    txManager,
	}
}
