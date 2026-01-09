package crawljob

import (
	"distributed-crawler/internal/application/service"
	crawljobrepo "distributed-crawler/internal/domain/crawl/repos/crawl_job"
	crawltaskrepo "distributed-crawler/internal/domain/crawl/repos/crawl_task"
	"distributed-crawler/internal/domain/crawl/repos/outbox"
	"distributed-crawler/internal/infra/persistence"
)

type crawlJobServ struct {
	crawlJobRepo  crawljobrepo.CrawlJobRepository
	crawlTaskRepo crawltaskrepo.CrawlTaskRepository
	outboxRepo    outbox.OutboxRepository
	txManager     persistence.TxManager
}

func NewService(
	crawlJobRepo crawljobrepo.CrawlJobRepository,
	crawlTaskRepo crawltaskrepo.CrawlTaskRepository,
	outboxRepo outbox.OutboxRepository,
	txManager persistence.TxManager,
) service.CrawlJobService {
	return &crawlJobServ{
		crawlJobRepo:  crawlJobRepo,
		crawlTaskRepo: crawlTaskRepo,
		outboxRepo:    outboxRepo,
		txManager:     txManager,
	}
}
