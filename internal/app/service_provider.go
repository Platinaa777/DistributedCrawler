package app

import (
	"context"
	"log"

	crawljob "distributed-crawler/internal/api/crawl_job"
	previewapi "distributed-crawler/internal/api/preview"
	"distributed-crawler/internal/application/service"
	crawljobservice "distributed-crawler/internal/application/service/crawl_job"
	crawltaskservice "distributed-crawler/internal/application/service/crawl_task"
	previewservice "distributed-crawler/internal/application/service/preview"
	"distributed-crawler/internal/config"
	"distributed-crawler/internal/config/env"
	"distributed-crawler/internal/domain/crawl/models"
	crawljobrepo "distributed-crawler/internal/domain/crawl/repos/crawl_job"
	crawljobconfig "distributed-crawler/internal/domain/crawl/repos/crawl_job_config"
	crawltaskrepo "distributed-crawler/internal/domain/crawl/repos/crawl_task"
	"distributed-crawler/internal/domain/crawl/repos/outbox"
	previewrepo "distributed-crawler/internal/domain/crawl/repos/preview"
	"distributed-crawler/internal/domain/crawl/services"
	"distributed-crawler/internal/infra/messaging/rabbitmq"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/infra/persistence/postgres/pg"
	crawljobrepoimpl "distributed-crawler/internal/infra/persistence/postgres/repos"
	"distributed-crawler/internal/infra/persistence/postgres/transaction"
	"distributed-crawler/internal/infra/services/contentstore"
	"distributed-crawler/internal/infra/services/fetcher"
	"distributed-crawler/internal/infra/services/sanitizer"
	"distributed-crawler/internal/worker"

	"go.uber.org/zap"
)

type serviceProvider struct {
	pgConfig       config.PGConfig
	grpcConfig     config.GRPCConfig
	httpConfig     config.HTTPConfig
	rabbitmqConfig config.RabbitMQConfig
	minioConfig    config.MinIOConfig

	dbClient           persistence.Client
	txManager          persistence.TxManager
	crawlJobRepo       crawljobrepo.CrawlJobRepository
	crawlJobConfigRepo crawljobconfig.CrawlJobConfigRepository
	crawlTaskRepo      crawltaskrepo.CrawlTaskRepository
	outboxRepo         outbox.OutboxRepository
	previewRepo        previewrepo.PreviewRepository
	rmqClient          rabbitmq.Client

	fetcher      services.Fetcher
	contentStore services.ContentStore
	htmlSanitizer previewservice.HTMLSanitizer

	crawlJobService  service.CrawlJobService
	crawlTaskService service.CrawlTaskService
	previewService   service.PreviewService

	crawlerServiceImpl *crawljob.CrawlJobImplementation
	previewServiceImpl *previewapi.PreviewImplementation
	outboxPublisher    *worker.OutboxPublisher

	logger *zap.Logger
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

func (s *serviceProvider) RabbitMQConfig() config.RabbitMQConfig {
	if s.rabbitmqConfig == nil {
		cfg, err := env.NewRabbitMQConfig()
		if err != nil {
			log.Fatalf("failed to get rabbitmq config: %s", err.Error())
		}

		s.rabbitmqConfig = cfg
	}

	return s.rabbitmqConfig
}

func (s *serviceProvider) Logger() *zap.Logger {
	if s.logger == nil {
		logger, err := zap.NewProduction()
		if err != nil {
			log.Fatalf("failed to create logger: %s", err.Error())
		}
		s.logger = logger
	}

	return s.logger
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

func (s *serviceProvider) CrawlJobConfigRepository(ctx context.Context) crawljobconfig.CrawlJobConfigRepository {
	if s.crawlJobConfigRepo == nil {
		s.crawlJobConfigRepo = crawljobrepoimpl.NewCrawlJobConfigRepository(s.DBClient(ctx))
	}

	return s.crawlJobConfigRepo
}

func (s *serviceProvider) CrawlJobService(ctx context.Context) service.CrawlJobService {
	if s.crawlJobService == nil {
		s.crawlJobService = crawljobservice.NewService(
			s.CrawlJobRepository(ctx),
			s.CrawlJobConfigRepository(ctx),
			s.CrawlTaskRepository(ctx),
			s.OutboxRepository(ctx),
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

func (s *serviceProvider) OutboxRepository(ctx context.Context) outbox.OutboxRepository {
	if s.outboxRepo == nil {
		s.outboxRepo = crawljobrepoimpl.NewOutboxRepository(s.DBClient(ctx))
	}

	return s.outboxRepo
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

func (s *serviceProvider) RabbitMQClient() rabbitmq.Client {
	if s.rmqClient == nil {
		client, err := rabbitmq.NewClient(s.RabbitMQConfig().URL())
		if err != nil {
			log.Fatalf("failed to create rabbitmq client: %v", err)
		}
		s.rmqClient = client
	}

	return s.rmqClient
}

func (s *serviceProvider) OutboxPublisher(ctx context.Context) *worker.OutboxPublisher {
	if s.outboxPublisher == nil {
		s.outboxPublisher = worker.NewOutboxPublisher(
			s.OutboxRepository(ctx),
			s.TxManager(ctx),
			s.RabbitMQClient(),
			s.RabbitMQConfig().GetQueueName(config.CrawlQueueKey),
			s.Logger(),
		)
	}

	return s.outboxPublisher
}

func (s *serviceProvider) MinIOConfig() config.MinIOConfig {
	if s.minioConfig == nil {
		cfg, err := env.NewMinIOConfig()
		if err != nil {
			log.Fatalf("failed to get minio config: %s", err.Error())
		}

		s.minioConfig = cfg
	}

	return s.minioConfig
}

func (s *serviceProvider) ContentStore() services.ContentStore {
	if s.contentStore == nil {
		minioCfg := s.MinIOConfig()
		store, err := contentstore.NewMinIOStore(
			minioCfg.Endpoint(),
			minioCfg.AccessKeyID(),
			minioCfg.SecretAccessKey(),
			minioCfg.UseSSL(),
			minioCfg.BucketName(),
			s.Logger(),
		)
		if err != nil {
			log.Fatalf("failed to create minio store: %v", err)
		}

		s.contentStore = store
	}

	return s.contentStore
}

func (s *serviceProvider) Fetcher() services.Fetcher {
	if s.fetcher == nil {
		// Use default auth and retry options for preview fetching
		authOptions := models.AuthOptions{}
		retryPolicy := models.RetryPolicy{
			MaxAttempts:        3,
			BackoffInitialMs:   1000,
			BackoffMultiplier:  2.0,
		}
		s.fetcher = fetcher.NewHTTPFetcher(authOptions, retryPolicy)
	}

	return s.fetcher
}

func (s *serviceProvider) HTMLSanitizer() previewservice.HTMLSanitizer {
	if s.htmlSanitizer == nil {
		s.htmlSanitizer = sanitizer.NewHTMLSanitizer()
	}

	return s.htmlSanitizer
}

func (s *serviceProvider) PreviewRepository(ctx context.Context) previewrepo.PreviewRepository {
	if s.previewRepo == nil {
		s.previewRepo = crawljobrepoimpl.NewPreviewRepository(s.DBClient(ctx))
	}

	return s.previewRepo
}

func (s *serviceProvider) PreviewService(ctx context.Context) service.PreviewService {
	if s.previewService == nil {
		// MinIOStore implements both ContentStore and PresignedURLGenerator
		minioStore := s.ContentStore().(*contentstore.MinIOStore)

		s.previewService = previewservice.NewService(
			s.PreviewRepository(ctx),
			s.Fetcher(),
			s.ContentStore(),
			s.HTMLSanitizer(),
			minioStore, // PresignedURLGenerator
			s.TxManager(ctx),
		)
	}

	return s.previewService
}

func (s *serviceProvider) PreviewServiceImpl(ctx context.Context) *previewapi.PreviewImplementation {
	if s.previewServiceImpl == nil {
		s.previewServiceImpl = previewapi.NewImplementation(
			s.PreviewService(ctx),
		)
	}

	return s.previewServiceImpl
}

func (s *serviceProvider) Close() error {
	if s.rmqClient != nil {
		if err := s.rmqClient.Close(); err != nil {
			log.Printf("failed to close rabbitmq client: %v", err)
		}
	}
	if s.dbClient != nil {
		return s.dbClient.Close()
	}
	if s.logger != nil {
		s.logger.Sync()
	}
	return nil
}
