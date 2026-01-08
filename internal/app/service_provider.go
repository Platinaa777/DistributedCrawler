package app

import (
	"context"
	"log"

	crawljob "distributed-crawler/internal/api/crawl_job"
	"distributed-crawler/internal/application/service"
	crawljobservice "distributed-crawler/internal/application/service/crawl_job"
	crawltaskservice "distributed-crawler/internal/application/service/crawl_task"
	"distributed-crawler/internal/config"
	"distributed-crawler/internal/config/env"
	crawljobrepo "distributed-crawler/internal/domain/crawl/repos/crawl_job"
	crawltaskrepo "distributed-crawler/internal/domain/crawl/repos/crawl_task"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/infra/persistence/postgres/pg"
	crawljobrepoimpl "distributed-crawler/internal/infra/persistence/postgres/repos"
	"distributed-crawler/internal/infra/persistence/postgres/transaction"
)

type serviceProvider struct {
	pgConfig   config.PGConfig
	grpcConfig config.GRPCConfig
	httpConfig config.HTTPConfig

	dbClient        persistence.Client
	txManager       persistence.TxManager
	crawlJobRepo    crawljobrepo.CrawlJobRepository
	crawlTaskRepo   crawltaskrepo.CrawlTaskRepository

	crawlJobService  service.CrawlJobService
	crawlTaskService service.CrawlTaskService

	crawlerServiceImpl *crawljob.CrawlJobImplementation
}

func newServiceProvider() *serviceProvider {
	return &serviceProvider{}
}

func (s *serviceProvider) PGConfig() config.PGConfig {
	if s.pgConfig == nil {
		cfg, err := env.NewPGConfig()
		if err != nil {
			log.Fatalf("failed to get pg config: %s", err.Error())
		}

		s.pgConfig = cfg
	}

	return s.pgConfig
}

func (s *serviceProvider) GRPCConfig() config.GRPCConfig {
	if s.grpcConfig == nil {
		cfg, err := env.NewGrpcConfig()
		if err != nil {
			log.Fatalf("failed to get grpc config: %s", err.Error())
		}

		s.grpcConfig = cfg
	}

	return s.grpcConfig
}

func (s *serviceProvider) HTTPConfig() config.HTTPConfig {
	if s.httpConfig == nil {
		cfg, err := env.NewHTTPConfig()
		if err != nil {
			log.Fatalf("failed to get http config: %s", err.Error())
		}

		s.httpConfig = cfg
	}

	return s.httpConfig
}

func (s *serviceProvider) DBClient(ctx context.Context) persistence.Client {
	if s.dbClient == nil {
		cl, err := pg.New(ctx, s.PGConfig().DSN())
		if err != nil {
			log.Fatalf("failed to create db client: %v", err)
		}

		err = cl.DB().Ping(ctx)
		if err != nil {
			log.Fatalf("ping error: %s", err.Error())
		}

		s.dbClient = cl
	}

	return s.dbClient
}

func (s *serviceProvider) TxManager(ctx context.Context) persistence.TxManager {
	if s.txManager == nil {
		s.txManager = transaction.NewTransactorManager(s.DBClient(ctx).DB())
	}

	return s.txManager
}

func (s *serviceProvider) CrawlJobRepository(ctx context.Context) crawljobrepo.CrawlJobRepository {
	if s.crawlJobRepo == nil {
		s.crawlJobRepo = crawljobrepoimpl.NewCrawlRepository(s.DBClient(ctx))
	}

	return s.crawlJobRepo
}

func (s *serviceProvider) CrawlJobService(ctx context.Context) service.CrawlJobService {
	if s.crawlJobService == nil {
		s.crawlJobService = crawljobservice.NewService(
			s.CrawlJobRepository(ctx),
			s.TxManager(ctx),
		)
	}

	return s.crawlJobService
}

func (s *serviceProvider) CrawlTaskRepository(ctx context.Context) crawltaskrepo.CrawlTaskRepository {
	if s.crawlTaskRepo == nil {
		s.crawlTaskRepo = crawljobrepoimpl.NewCrawlTaskRepository(s.DBClient(ctx))
	}

	return s.crawlTaskRepo
}

func (s *serviceProvider) CrawlTaskService(ctx context.Context) service.CrawlTaskService {
	if s.crawlTaskService == nil {
		s.crawlTaskService = crawltaskservice.NewCrawlTaskService(
			s.CrawlTaskRepository(ctx),
		)
	}

	return s.crawlTaskService
}

func (s *serviceProvider) CrawlerServiceImpl(ctx context.Context) *crawljob.CrawlJobImplementation {
	if s.crawlerServiceImpl == nil {
		s.crawlerServiceImpl = crawljob.NewImplementation(
			s.CrawlJobService(ctx),
			s.CrawlTaskService(ctx),
		)
	}

	return s.crawlerServiceImpl
}

func (s *serviceProvider) Close() error {
	if s.dbClient != nil {
		return s.dbClient.Close()
	}
	return nil
}
