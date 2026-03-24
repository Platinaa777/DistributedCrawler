package app

import (
	"context"
	"log"
	"time"

	authapi "distributed-crawler/internal/api/auth"
	crawljob "distributed-crawler/internal/api/crawl_job"
	previewapi "distributed-crawler/internal/api/preview"
	queueadminapi "distributed-crawler/internal/api/queue_admin"
	userapi "distributed-crawler/internal/api/user"
	workerapi "distributed-crawler/internal/api/worker"
	"distributed-crawler/internal/application/service"
	authservice "distributed-crawler/internal/application/service/auth"
	crawljobservice "distributed-crawler/internal/application/service/crawl_job"
	crawltaskservice "distributed-crawler/internal/application/service/crawl_task"
	previewservice "distributed-crawler/internal/application/service/preview"
	userservice "distributed-crawler/internal/application/service/user"
	appqueue "distributed-crawler/internal/application/queue"
	"distributed-crawler/internal/auth"
	"distributed-crawler/internal/config"
	"distributed-crawler/internal/config/env"
	authmodels "distributed-crawler/internal/domain/auth/models"
	refreshtokenrepo "distributed-crawler/internal/domain/auth/repos/refresh_token"
	userrepo "distributed-crawler/internal/domain/auth/repos/user"
	"distributed-crawler/internal/domain/crawl/models"
	crawljobrepo "distributed-crawler/internal/domain/crawl/repos/crawl_job"
	crawljobconfig "distributed-crawler/internal/domain/crawl/repos/crawl_job_config"
	crawltaskrepo "distributed-crawler/internal/domain/crawl/repos/crawl_task"
	"distributed-crawler/internal/domain/crawl/repos/outbox"
	previewrepo "distributed-crawler/internal/domain/crawl/repos/preview"
	"distributed-crawler/internal/domain/crawl/services"
	queuerepos "distributed-crawler/internal/domain/queue/repos"
	"distributed-crawler/internal/infra/messaging"
	kafkaclient "distributed-crawler/internal/infra/messaging/kafka"
	memorybroker "distributed-crawler/internal/infra/messaging/memory/broker"
	rabbitmqclient "distributed-crawler/internal/infra/messaging/rabbitmq"
	"distributed-crawler/internal/infra/secrets"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/worker/routing"

	"github.com/redis/go-redis/v9"
	"distributed-crawler/internal/infra/persistence/postgres/pg"
	crawljobrepoimpl "distributed-crawler/internal/infra/persistence/postgres/repos"
	"distributed-crawler/internal/infra/persistence/postgres/transaction"
	"distributed-crawler/internal/infra/services/contentstore"
	"distributed-crawler/internal/infra/services/fetcher"
	"distributed-crawler/internal/infra/services/sanitizer"
	"distributed-crawler/internal/telemetry"
	"distributed-crawler/internal/worker"
	"distributed-crawler/internal/workerhealth"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type serviceProvider struct {
	pgConfig       config.PGConfig
	grpcConfig     config.GRPCConfig
	httpConfig     config.HTTPConfig
	rabbitmqConfig config.RabbitMQConfig
	kafkaConfig    config.KafkaConfig
	minioConfig    config.MinIOConfig
	authConfig     config.AuthConfig
	otelConfig     config.OTelConfig

	telemetryProvider *telemetry.TelemetryProvider
	metrics           *telemetry.Metrics

	dbClient           persistence.Client
	txManager          persistence.TxManager
	crawlJobRepo       crawljobrepo.CrawlJobRepository
	crawlJobConfigRepo crawljobconfig.CrawlJobConfigRepository
	crawlTaskRepo      crawltaskrepo.CrawlTaskRepository
	outboxRepo         outbox.OutboxRepository
	previewRepo        previewrepo.PreviewRepository
	userRepo           userrepo.UserRepository
	refreshTokenRepo   refreshtokenrepo.RefreshTokenRepository
	msgClient          messaging.Client

	fetcher               services.Fetcher
	previewFetcherFactory services.FetcherFactory
	contentStore          services.ContentStore
	htmlSanitizer         previewservice.HTMLSanitizer
	jwtService            *auth.JWTService

	crawlJobService  service.CrawlJobService
	crawlTaskService service.CrawlTaskService
	previewService   service.PreviewService
	authService      service.AuthService
	userService      service.UserService

	crawlerServiceImpl  *crawljob.CrawlJobImplementation
	previewServiceImpl  *previewapi.PreviewImplementation
	authServiceImpl     *authapi.AuthImplementation
	userServiceImpl     *userapi.UserImplementation
	workerServiceImpl   *workerapi.WorkerImplementation
	queueAdminImpl      *queueadminapi.QueueAdminImplementation
	outboxPublisher     *worker.OutboxPublisher
	scheduleWorker      *worker.ScheduleWorker
	workerRegistry      *workerhealth.Registry

	queueEndpointRepo  queuerepos.QueueEndpointRepository
	queueRuleRepo      queuerepos.QueueRoutingRuleRepository
	queueService       *appqueue.Service
	secretsStore       secrets.SecretsStore

	redisClient   *redis.Client
	routingPolicy routing.QueueRoutingPolicy

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

func (s *serviceProvider) KafkaConfig() config.KafkaConfig {
	if s.kafkaConfig == nil {
		cfg, err := env.NewKafkaConfig()
		if err != nil {
			log.Fatalf("failed to get kafka config: %s", err.Error())
		}

		s.kafkaConfig = cfg
	}

	return s.kafkaConfig
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
		var cl persistence.Client
		var err error

		if s.PGConfig().ShardingEnabled() {
			cl, err = pg.NewSharded(ctx, s.PGConfig().ShardDSNs())
		} else {
			cl, err = pg.New(ctx, s.PGConfig().DSN())
		}
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
			s.Metrics(ctx),
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
		// MinIOStore implements PresignedURLGenerator interface
		minioStore := s.ContentStore().(*contentstore.MinIOStore)
		s.crawlerServiceImpl = crawljob.NewImplementation(
			s.CrawlJobService(ctx),
			s.CrawlTaskService(ctx),
			minioStore,
		)
	}

	return s.crawlerServiceImpl
}

func (s *serviceProvider) MessagingClient() messaging.Client {
	if s.msgClient == nil {
		var client messaging.Client
		var err error

		switch env.GetBrokerType() {
		case env.BrokerKafka:
			kafkaCfg := s.KafkaConfig()
			client, err = kafkaclient.NewClient(kafkaCfg.Brokers(), kafkaCfg.ConsumerGroup())
			if err != nil {
				log.Fatalf("failed to create kafka client: %v", err)
			}
			log.Printf("messaging: using Kafka broker (brokers=%v, group=%s)", kafkaCfg.Brokers(), kafkaCfg.ConsumerGroup())

		case env.BrokerGRPCMemory:
			mbCfg, cfgErr := env.NewMemoryBrokerConfig()
			if cfgErr != nil {
				log.Fatalf("failed to get memory broker config: %v", cfgErr)
			}
			client, err = memorybroker.NewGRPCClient(mbCfg.Address())
			if err != nil {
				log.Fatalf("failed to connect to gRPC memory broker: %v", err)
			}
			log.Printf("messaging: using gRPC memory broker at %s", mbCfg.Address())

		default:
			client, err = rabbitmqclient.NewClient(s.RabbitMQConfig().URL())
			if err != nil {
				log.Fatalf("failed to create rabbitmq client: %v", err)
			}
			log.Printf("messaging: using RabbitMQ broker")
		}

		s.msgClient = client
	}

	return s.msgClient
}

func (s *serviceProvider) getQueueName(key string) string {
	switch env.GetBrokerType() {
	case env.BrokerKafka:
		return s.KafkaConfig().GetTopicName(key)
	case env.BrokerGRPCMemory:
		return key // memory broker uses the key directly
	default:
		return s.RabbitMQConfig().GetQueueName(key)
	}
}

// RedisClient returns a Redis client if REDIS_ADDRESS is configured, otherwise nil.
func (s *serviceProvider) RedisClient() *redis.Client {
	if s.redisClient == nil {
		cfg, err := env.NewRedisConfig()
		if err != nil {
			// Redis not configured — routing will fall back to in-memory
			return nil
		}
		s.redisClient = redis.NewClient(&redis.Options{
			Addr:     cfg.Address(),
			Password: cfg.Password(),
			DB:       cfg.DB(),
		})
	}
	return s.redisClient
}

// RoutingPolicy returns a job-aware queue routing policy.
// Uses Redis round-robin if Redis is available, otherwise weighted random in-memory.
func (s *serviceProvider) RoutingPolicy(ctx context.Context) routing.QueueRoutingPolicy {
	if s.routingPolicy == nil {
		loader := routing.NewDBJobQueueLoader(s.DBClient(ctx).DB())
		fallbackCrawl := s.getQueueName(config.CrawlQueueKey)
		fallbackParse := s.getQueueName(config.ParsingQueueKey)

		rdb := s.RedisClient()
		if rdb != nil {
			s.routingPolicy = routing.NewRedisRoutingPolicy(loader, rdb, fallbackCrawl, fallbackParse)
		} else {
			s.routingPolicy = routing.NewInMemoryRoutingPolicy(loader, fallbackCrawl, fallbackParse)
		}
	}
	return s.routingPolicy
}

func (s *serviceProvider) OutboxPublisher(ctx context.Context) *worker.OutboxPublisher {
	if s.outboxPublisher == nil {
		var tracer trace.Tracer
		tp := s.TelemetryProvider(ctx)
		if tp != nil {
			tracer = tp.Tracer("outbox-publisher")
		}

		s.outboxPublisher = worker.NewOutboxPublisher(
			s.OutboxRepository(ctx),
			s.TxManager(ctx),
			s.MessagingClient(),
			s.getQueueName(config.CrawlQueueKey),
			s.Logger(),
			tracer,
		).WithRoutingPolicy(s.RoutingPolicy(ctx))
	}

	return s.outboxPublisher
}

func (s *serviceProvider) ScheduleWorker(ctx context.Context) *worker.ScheduleWorker {
	if s.scheduleWorker == nil {
		s.scheduleWorker = worker.NewScheduleWorker(
			s.CrawlJobRepository(ctx),
			s.CrawlJobConfigRepository(ctx),
			s.CrawlTaskRepository(ctx),
			s.OutboxRepository(ctx),
			s.TxManager(ctx),
			s.Logger(),
			s.Metrics(ctx),
		)
	}
	return s.scheduleWorker
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
			minioCfg.PublicBaseURL(),
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
			MaxAttempts:       3,
			BackoffInitialMs:  1000,
			BackoffMultiplier: 2.0,
		}
		s.fetcher = fetcher.NewBrowserFetcher(authOptions, retryPolicy, "")
	}

	return s.fetcher
}

func (s *serviceProvider) PreviewFetcherFactory() services.FetcherFactory {
	if s.previewFetcherFactory == nil {
		s.previewFetcherFactory = fetcher.NewBrowserFetcherFactory("")
	}

	return s.previewFetcherFactory
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
		retryPolicy := models.RetryPolicy{
			MaxAttempts:       3,
			BackoffInitialMs:  1000,
			BackoffMultiplier: 2.0,
		}

		s.previewService = previewservice.NewService(
			s.PreviewRepository(ctx),
			s.PreviewFetcherFactory(),
			s.ContentStore(),
			s.HTMLSanitizer(),
			minioStore, // PresignedURLGenerator
			s.TxManager(ctx),
			retryPolicy,
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

func (s *serviceProvider) AuthConfig() config.AuthConfig {
	if s.authConfig == nil {
		cfg, err := env.NewAuthConfig()
		if err != nil {
			log.Fatalf("failed to get auth config: %s", err.Error())
		}

		s.authConfig = cfg
	}

	return s.authConfig
}

func (s *serviceProvider) OTelConfig() config.OTelConfig {
	if s.otelConfig == nil {
		cfg, err := env.NewOTelConfig()
		if err != nil {
			log.Printf("failed to get otel config: %s", err.Error())
			return nil
		}
		s.otelConfig = cfg
	}
	return s.otelConfig
}

func (s *serviceProvider) TelemetryProvider(ctx context.Context) *telemetry.TelemetryProvider {
	if s.telemetryProvider == nil {
		cfg := s.OTelConfig()
		if cfg == nil || !cfg.Enabled() {
			return nil
		}
		tp, err := telemetry.NewTelemetryProvider(ctx, cfg)
		if err != nil {
			log.Printf("failed to create telemetry provider: %v", err)
			return nil
		}
		s.telemetryProvider = tp
	}
	return s.telemetryProvider
}

func (s *serviceProvider) Metrics(ctx context.Context) *telemetry.Metrics {
	if s.metrics == nil {
		tp := s.TelemetryProvider(ctx)
		if tp == nil {
			return nil
		}
		m, err := telemetry.NewMetrics(tp.Meter("distributed-crawler"))
		if err != nil {
			log.Printf("failed to create metrics: %v", err)
			return nil
		}
		s.metrics = m
	}
	return s.metrics
}

func (s *serviceProvider) UserRepository(ctx context.Context) userrepo.UserRepository {
	if s.userRepo == nil {
		s.userRepo = crawljobrepoimpl.NewUserRepository(s.DBClient(ctx))
	}

	return s.userRepo
}

func (s *serviceProvider) RefreshTokenRepository(ctx context.Context) refreshtokenrepo.RefreshTokenRepository {
	if s.refreshTokenRepo == nil {
		s.refreshTokenRepo = crawljobrepoimpl.NewRefreshTokenRepository(s.DBClient(ctx))
	}

	return s.refreshTokenRepo
}

func (s *serviceProvider) JWTService() *auth.JWTService {
	if s.jwtService == nil {
		authCfg := s.AuthConfig()
		s.jwtService = auth.NewJWTService(
			authCfg.JWTSecret(),
			authCfg.Issuer(),
			authCfg.Audience(),
		)
	}

	return s.jwtService
}

func (s *serviceProvider) AuthService(ctx context.Context) service.AuthService {
	if s.authService == nil {
		authCfg := s.AuthConfig()

		accessTokenTTL, err := time.ParseDuration(authCfg.AccessTokenTTL())
		if err != nil {
			log.Fatalf("failed to parse access token TTL: %v", err)
		}

		refreshTokenTTL, err := time.ParseDuration(authCfg.RefreshTokenTTL())
		if err != nil {
			log.Fatalf("failed to parse refresh token TTL: %v", err)
		}

		s.authService = authservice.NewAuthService(
			s.UserRepository(ctx),
			s.RefreshTokenRepository(ctx),
			s.TxManager(ctx),
			s.JWTService(),
			accessTokenTTL,
			refreshTokenTTL,
		)
	}

	return s.authService
}

func (s *serviceProvider) UserService(ctx context.Context) service.UserService {
	if s.userService == nil {
		s.userService = userservice.NewUserService(
			s.UserRepository(ctx),
			s.TxManager(ctx),
		)
	}

	return s.userService
}

func (s *serviceProvider) AuthServiceImpl(ctx context.Context) *authapi.AuthImplementation {
	if s.authServiceImpl == nil {
		s.authServiceImpl = authapi.NewImplementation(
			s.AuthService(ctx),
		)
	}

	return s.authServiceImpl
}

func (s *serviceProvider) UserServiceImpl(ctx context.Context) *userapi.UserImplementation {
	if s.userServiceImpl == nil {
		s.userServiceImpl = userapi.NewImplementation(
			s.UserService(ctx),
		)
	}

	return s.userServiceImpl
}

func (s *serviceProvider) WorkerRegistry() *workerhealth.Registry {
	if s.workerRegistry == nil {
		s.workerRegistry = workerhealth.NewRegistry(12*time.Second, 1*time.Minute)
	}

	return s.workerRegistry
}

func (s *serviceProvider) WorkerServiceImpl(_ context.Context) *workerapi.WorkerImplementation {
	if s.workerServiceImpl == nil {
		s.workerServiceImpl = workerapi.NewImplementation(
			s.WorkerRegistry(),
			s.Logger(),
		)
	}

	return s.workerServiceImpl
}

func (s *serviceProvider) EnsureDefaultAdmin(ctx context.Context) error {
	authCfg := s.AuthConfig()
	email := authCfg.DefaultUserEmail()
	password := authCfg.DefaultUserPassword()

	user, err := s.UserRepository(ctx).GetByEmail(ctx, email)
	if err != nil {
		return err
	}

	if user != nil {
		if user.Role != authmodels.RoleAdministrator {
			return s.UserRepository(ctx).UpdateRole(ctx, user.ID, authmodels.RoleAdministrator)
		}
		return nil
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}

	admin := authmodels.NewUserWithRole(email, passwordHash, authmodels.RoleAdministrator)
	_, err = s.UserRepository(ctx).Create(ctx, admin)
	return err
}

func (s *serviceProvider) QueueEndpointRepo(ctx context.Context) queuerepos.QueueEndpointRepository {
	if s.queueEndpointRepo == nil {
		s.queueEndpointRepo = crawljobrepoimpl.NewQueueEndpointRepository(s.DBClient(ctx))
	}
	return s.queueEndpointRepo
}

func (s *serviceProvider) QueueRuleRepo(ctx context.Context) queuerepos.QueueRoutingRuleRepository {
	if s.queueRuleRepo == nil {
		s.queueRuleRepo = crawljobrepoimpl.NewQueueRoutingRuleRepository(s.DBClient(ctx))
	}
	return s.queueRuleRepo
}

func (s *serviceProvider) QueueService(ctx context.Context) *appqueue.Service {
	if s.queueService == nil {
		s.queueService = appqueue.NewService(
			s.QueueEndpointRepo(ctx),
			s.QueueRuleRepo(ctx),
		)
	}
	return s.queueService
}

func (s *serviceProvider) QueueAdminImpl(ctx context.Context) *queueadminapi.QueueAdminImplementation {
	if s.queueAdminImpl == nil {
		s.queueAdminImpl = queueadminapi.NewImplementation(s.QueueService(ctx))
	}
	return s.queueAdminImpl
}

// SecretsStore returns the initialized secrets store, or nil if not configured.
func (s *serviceProvider) SecretsStore() secrets.SecretsStore {
	return s.secretsStore
}

// InitSecretsStore initializes the file-based secrets store using appCtx for the
// background reload goroutine. It is optional: if QUEUE_SECRETS_FILE_PATH is not
// set the store is left nil and callers must handle that gracefully.
func (s *serviceProvider) InitSecretsStore(appCtx context.Context) {
	secretsCfg, err := env.NewSecretsConfig()
	if err != nil {
		log.Printf("secrets store disabled: %v", err)
		return
	}

	reloadInterval := secretsCfg.ReloadInterval()
	if !secretsCfg.WatchEnabled() {
		reloadInterval = 24 * time.Hour
	}

	store, err := secrets.NewFileSecretsStore(appCtx, secretsCfg.FilePath(), reloadInterval)
	if err != nil {
		log.Printf("failed to initialize secrets store from %s: %v", secretsCfg.FilePath(), err)
		return
	}

	s.secretsStore = store
	log.Printf("secrets store initialized: path=%s watch=%v interval=%s",
		secretsCfg.FilePath(), secretsCfg.WatchEnabled(), reloadInterval)
}

func (s *serviceProvider) Close() error {
	// Shutdown telemetry first to flush pending data
	if s.telemetryProvider != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.telemetryProvider.Shutdown(ctx); err != nil {
			log.Printf("failed to shutdown telemetry: %v", err)
		}
	}
	if s.msgClient != nil {
		if err := s.msgClient.Close(); err != nil {
			log.Printf("failed to close messaging client: %v", err)
		}
	}
	if s.redisClient != nil {
		if err := s.redisClient.Close(); err != nil {
			log.Printf("failed to close redis client: %v", err)
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
