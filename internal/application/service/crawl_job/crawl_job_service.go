package crawljob

import (
	"distributed-crawler/internal/application/service"
	crawljobrepo "distributed-crawler/internal/domain/crawl/repos/crawl_job"
	crawljobconfigrepo "distributed-crawler/internal/domain/crawl/repos/crawl_job_config"
	crawltaskrepo "distributed-crawler/internal/domain/crawl/repos/crawl_task"
	"distributed-crawler/internal/domain/crawl/repos/outbox"
	"distributed-crawler/internal/infra/persistence"
)

type crawlJobServ struct {
	crawlJobRepo       crawljobrepo.CrawlJobRepository
	crawlJobConfigRepo crawljobconfigrepo.CrawlJobConfigRepository
	crawlTaskRepo      crawltaskrepo.CrawlTaskRepository
	outboxRepo         outbox.OutboxRepository
	txManager          persistence.TxManager
}

func NewService(
	crawlJobRepo crawljobrepo.CrawlJobRepository,
	crawlJobConfigRepo crawljobconfigrepo.CrawlJobConfigRepository,
	crawlTaskRepo crawltaskrepo.CrawlTaskRepository,
	outboxRepo outbox.OutboxRepository,
	txManager persistence.TxManager,
) service.CrawlJobService {
	return &crawlJobServ{
		crawlJobRepo:       crawlJobRepo,
		crawlJobConfigRepo: crawlJobConfigRepo,
		crawlTaskRepo:      crawlTaskRepo,
		outboxRepo:         outboxRepo,
		txManager:          txManager,
	}
}
