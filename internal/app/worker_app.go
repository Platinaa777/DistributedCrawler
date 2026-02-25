package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap/zapcore"

	"distributed-crawler/internal/config"
	"distributed-crawler/internal/config/env"
	"distributed-crawler/internal/infra/cache"
	"distributed-crawler/internal/infra/logger"
	"distributed-crawler/internal/infra/messaging"
	kafkaclient "distributed-crawler/internal/infra/messaging/kafka"
	memorybroker "distributed-crawler/internal/infra/messaging/memory/broker"
	rabbitmqclient "distributed-crawler/internal/infra/messaging/rabbitmq"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/infra/persistence/postgres/pg"
	"distributed-crawler/internal/infra/persistence/postgres/repos"
	"distributed-crawler/internal/infra/persistence/postgres/transaction"
	"distributed-crawler/internal/infra/services/contentstore"
	"distributed-crawler/internal/infra/services/fetcher"
	"distributed-crawler/internal/telemetry"
	"distributed-crawler/internal/worker"
	crawlergrpc "distributed-crawler/pkg/v1"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var workerConfigPath string

func init() {
	flag.StringVar(&workerConfigPath, "worker-config-path", ".worker.env", "path to config file")
}

// WorkerType defines the type of worker to run
type WorkerType string

const (
	FetchWorkerType     WorkerType = "fetch"
	ParserWorkerType    WorkerType = "parser"
	ExportWorkerType    WorkerType = "export"
	SchedulerWorkerType WorkerType = "scheduler"
)

type WorkerApp struct {
	workerType  WorkerType
	zapLogger   *zap.Logger
	pgClient    persistence.Client
	msgClient   messaging.Client
	rmqConfig   config.RabbitMQConfig
	kafkaConfig config.KafkaConfig
	brokerType  string
	redisClient *redis.Client
	grpcConfig  config.GRPCConfig

	telemetryProvider *telemetry.TelemetryProvider
	metrics           *telemetry.Metrics

	workerID  string
	startedAt time.Time
	status    atomic.Int32

	fetchWorker    *worker.FetchWorker
	parserWorker   *worker.ParserWorker
	exportWorker   *worker.ExportWorker
	scheduleWorker *worker.ScheduleWorker

	workerCtx     context.Context
	workerCancel  context.CancelFunc
	consumeCtx    context.Context
	consumeCancel context.CancelFunc

	activeTasksCounter interface {
		ActiveTasks() int32
	}

	monitor   *worker.WorkerMonitor
	drainOnce sync.Once
}

// NewWorkerApp creates a new worker application
func NewWorkerApp(ctx context.Context, workerType WorkerType) (*WorkerApp, error) {
	a := &WorkerApp{
		workerType: workerType,
	}

	err := a.initDeps(ctx)
	if err != nil {
		return nil, err
	}

	return a, nil
}

// Run starts the worker application
func (a *WorkerApp) Run() error {
	defer func() {
		// Shutdown telemetry first to flush pending data
		if a.telemetryProvider != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := a.telemetryProvider.Shutdown(ctx); err != nil {
				a.zapLogger.Warn("Failed to shutdown telemetry", zap.Error(err))
			}
		}
		if a.pgClient != nil {
			a.pgClient.Close()
		}
		if a.msgClient != nil {
			a.msgClient.Close()
		}
		if a.redisClient != nil {
			a.redisClient.Close()
		}
		if a.zapLogger != nil {
			a.zapLogger.Sync()
		}
	}()

	wg := sync.WaitGroup{}

	a.initMonitor()

	wg.Go(func() {
		if a.monitor != nil {
			a.monitor.Run(a.workerCtx)
		}
	})

	wg.Go(func() {
		if err := a.runWorker(); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				a.zapLogger.Info("Worker stopped gracefully")
				return
			}
			a.zapLogger.Fatal("Worker failed", zap.Error(err))
		}
	})

	wg.Wait()

	return nil
}

func (a *WorkerApp) initDeps(ctx context.Context) error {
	inits := []func(context.Context) error{
		a.initConfig,
		a.initGRPCConfig,
		a.initLogger,
		a.initTelemetry,
		a.initPostgreSQL,
		a.initMessaging,
		a.initRedis,
		a.initWorker,
	}

	for _, f := range inits {
		err := f(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *WorkerApp) initConfig(_ context.Context) error {
	err := config.Load(workerConfigPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	return nil
}

func (a *WorkerApp) initGRPCConfig(_ context.Context) error {
	grpcCfg, err := env.NewGrpcConfig()
	if err != nil {
		log.Fatalf("failed to get grpc config: %v", err)
	}
	a.grpcConfig = grpcCfg
	return nil
}

func (a *WorkerApp) initLogger(_ context.Context) error {
	loggerConfig, err := env.NewLoggerConfig()
	if err != nil {
		log.Fatalf("failed to get logger config: %v", err)
	}

	var zapLogger *zap.Logger
	if loggerConfig.Env() == "production" {
		zapLogger, err = zap.NewProduction()
	} else {
		zapLogger, err = zap.NewDevelopment()
	}

	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}

	osCfg, osErr := env.NewOpenSearchConfig()
	if osErr == nil && osCfg.Enabled() {
		osCore := logger.NewOpenSearchCore(
			zapcore.DebugLevel,
			osCfg.Endpoint(),
			osCfg.Index(),
			osCfg.BatchSize(),
			osCfg.FlushInterval(),
		)
		tee := zapcore.NewTee(zapLogger.Core(), osCore)
		zapLogger = zapLogger.WithOptions(zap.WrapCore(func(zapcore.Core) zapcore.Core {
			return tee
		}))
	}

	a.zapLogger = zapLogger
	return nil
}

func (a *WorkerApp) initTelemetry(ctx context.Context) error {
	otelCfg, err := env.NewOTelConfig()
	if err != nil {
		a.zapLogger.Warn("Failed to get OTel config, telemetry disabled", zap.Error(err))
		return nil
	}

	if !otelCfg.Enabled() {
		a.zapLogger.Info("Telemetry is disabled")
		return nil
	}

	// Override service name with worker-type suffix for unique identification
	serviceName := fmt.Sprintf("distributed-crawler-worker-%s", a.workerType)
	otelCfg = env.WithServiceName(otelCfg, serviceName)

	tp, err := telemetry.NewTelemetryProvider(ctx, otelCfg)
	if err != nil {
		a.zapLogger.Warn("Failed to create telemetry provider", zap.Error(err))
		return nil
	}

	a.telemetryProvider = tp

	if tp != nil {
		m, err := telemetry.NewMetrics(tp.Meter(serviceName))
		if err != nil {
			a.zapLogger.Warn("Failed to create metrics", zap.Error(err))
		} else {
			a.metrics = m
		}
	}

	a.zapLogger.Info("Telemetry initialized successfully", zap.String("serviceName", serviceName))
	return nil
}

func (a *WorkerApp) initPostgreSQL(ctx context.Context) error {
	pgCfg, err := env.NewPGConfig()
	if err != nil {
		a.zapLogger.Fatal("Failed to create PG config", zap.Error(err))
	}

	var pgClient persistence.Client
	if pgCfg.ShardingEnabled() {
		pgClient, err = pg.NewSharded(ctx, pgCfg.ShardDSNs())
	} else {
		pgClient, err = pg.New(ctx, pgCfg.DSN())
	}
	if err != nil {
		a.zapLogger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}

	err = pgClient.DB().Ping(ctx)
	if err != nil {
		a.zapLogger.Fatal("Failed to ping PostgreSQL", zap.Error(err))
	}

	a.pgClient = pgClient
	return nil
}

func (a *WorkerApp) initMessaging(_ context.Context) error {
	a.brokerType = env.GetBrokerType()

	switch a.brokerType {
	case env.BrokerKafka:
		kafkaCfg, err := env.NewKafkaConfig()
		if err != nil {
			a.zapLogger.Fatal("Failed to create Kafka config", zap.Error(err))
		}
		client, err := kafkaclient.NewClient(kafkaCfg.Brokers(), kafkaCfg.ConsumerGroup())
		if err != nil {
			a.zapLogger.Fatal("Failed to create Kafka client", zap.Error(err))
		}
		a.kafkaConfig = kafkaCfg
		a.msgClient = client
		a.zapLogger.Info("Messaging: using Kafka broker",
			zap.Strings("brokers", kafkaCfg.Brokers()),
			zap.String("consumer_group", kafkaCfg.ConsumerGroup()),
		)

	case env.BrokerGRPCMemory:
		mbCfg, err := env.NewMemoryBrokerConfig()
		if err != nil {
			a.zapLogger.Fatal("Failed to create memory broker config", zap.Error(err))
		}
		client, err := memorybroker.NewGRPCClient(mbCfg.Address())
		if err != nil {
			a.zapLogger.Fatal("Failed to connect to gRPC memory broker", zap.Error(err))
		}
		a.msgClient = client
		a.zapLogger.Info("Messaging: using gRPC memory broker", zap.String("addr", mbCfg.Address()))

	default:
		rmqCfg, err := env.NewRabbitMQConfig()
		if err != nil {
			a.zapLogger.Fatal("Failed to create RabbitMQ config", zap.Error(err))
		}
		client, err := rabbitmqclient.NewClient(rmqCfg.URL())
		if err != nil {
			a.zapLogger.Fatal("Failed to create RabbitMQ client", zap.Error(err))
		}
		a.rmqConfig = rmqCfg
		a.msgClient = client
		a.zapLogger.Info("Messaging: using RabbitMQ broker")
	}

	return nil
}

func (a *WorkerApp) getQueueName(key string) string {
	if a.brokerType == env.BrokerKafka && a.kafkaConfig != nil {
		return a.kafkaConfig.GetTopicName(key)
	}
	if a.rmqConfig != nil {
		return a.rmqConfig.GetQueueName(key)
	}
	return key
}

func (a *WorkerApp) initRedis(_ context.Context) error {
	redisCfg, err := env.NewRedisConfig()
	if err != nil {
		a.zapLogger.Fatal("Failed to create Redis config", zap.Error(err))
	}

	redisClient, err := cache.NewRedisClient(redisCfg)
	if err != nil {
		a.zapLogger.Fatal("Failed to create Redis client", zap.Error(err))
	}

	a.redisClient = redisClient
	return nil
}

func (a *WorkerApp) initWorker(ctx context.Context) error {
	a.workerCtx, a.workerCancel = context.WithCancel(ctx)
	a.consumeCtx, a.consumeCancel = context.WithCancel(a.workerCtx)
	a.startedAt = time.Now().UTC()
	a.status.Store(int32(crawlergrpc.WorkerStatus_WORKER_STATUS_ACTIVE))

	switch a.workerType {
	case FetchWorkerType:
		return a.initFetchWorker()
	case ParserWorkerType:
		return a.initParserWorker()
	case ExportWorkerType:
		return a.initExportWorker()
	case SchedulerWorkerType:
		return a.initScheduleWorker()
	default:
		log.Fatalf("unknown worker type: %s", a.workerType)
	}

	return nil
}

func (a *WorkerApp) initFetchWorker() error {
	// Initialize MinIO
	minioCfg, err := env.NewMinIOConfig()
	if err != nil {
		a.zapLogger.Fatal("Failed to create MinIO config", zap.Error(err))
	}

	contentStore, err := contentstore.NewMinIOStore(
		minioCfg.Endpoint(),
		minioCfg.AccessKeyID(),
		minioCfg.SecretAccessKey(),
		minioCfg.UseSSL(),
		minioCfg.BucketName(),
		a.zapLogger,
	)
	if err != nil {
		a.zapLogger.Fatal("Failed to create MinIO store", zap.Error(err))
	}

	// Initialize repositories
	taskRepo := repos.NewCrawlTaskRepository(a.pgClient)
	jobConfigRepo := repos.NewCrawlJobConfigRepository(a.pgClient)

	// Initialize fetcher services
	fetcherFactory := fetcher.NewHTTPFetcherFactory()
	scopeValidator := fetcher.NewDomainScopeValidator()

	// Initialize rate limiter with 5 minute TTL.
	// "inmemory" stores buckets locally per worker process.
	var rateLimiterType = env.GetLimiterType()
	var rateLimiter = cache.NewRedisRateLimiter(a.redisClient, 5*time.Minute)
	if rateLimiterType == env.LimiterInMemory {
		rateLimiter = cache.NewInMemoryRateLimiter(5 * time.Minute)
	}
	a.zapLogger.Info("Rate limiting: using provider", zap.String("provider", rateLimiterType))

	// Initialize robots.txt service with 24 hour cache TTL
	robotsTxtService := cache.NewCachedRobotsTxtService(a.redisClient, 24*time.Hour, a.zapLogger)

	// Get queue/topic names from configuration
	crawlQueue := a.getQueueName(config.CrawlQueueKey)
	parsingQueue := a.getQueueName(config.ParsingQueueKey)

	// Get tracer if telemetry is available
	var tracer trace.Tracer
	if a.telemetryProvider != nil {
		tracer = a.telemetryProvider.Tracer("fetch-worker")
	}

	// Create fetch worker
	a.fetchWorker = worker.NewFetchWorker(
		a.msgClient,
		crawlQueue,   // Consume from crawl_queue
		parsingQueue, // Publish to parsing_queue
		contentStore,
		taskRepo,
		jobConfigRepo,
		fetcherFactory,
		scopeValidator,
		rateLimiter,
		robotsTxtService,
		a.zapLogger,
		tracer,
		a.metrics,
		a.workerID,
	)
	a.activeTasksCounter = a.fetchWorker

	return nil
}

func (a *WorkerApp) initParserWorker() error {
	// Initialize MinIO
	minioCfg, err := env.NewMinIOConfig()
	if err != nil {
		a.zapLogger.Fatal("Failed to create MinIO config", zap.Error(err))
	}

	contentStore, err := contentstore.NewMinIOStore(
		minioCfg.Endpoint(),
		minioCfg.AccessKeyID(),
		minioCfg.SecretAccessKey(),
		minioCfg.UseSSL(),
		minioCfg.BucketName(),
		a.zapLogger,
	)
	if err != nil {
		a.zapLogger.Fatal("Failed to create MinIO store", zap.Error(err))
	}

	// Initialize repositories
	taskRepo := repos.NewCrawlTaskRepository(a.pgClient)
	jobRepo := repos.NewCrawlRepository(a.pgClient)
	jobConfigRepo := repos.NewCrawlJobConfigRepository(a.pgClient)
	outboxRepo := repos.NewOutboxRepository(a.pgClient)

	// Initialize transaction manager
	txManager := transaction.NewTransactorManager(a.pgClient.DB())

	// Initialize scope validator
	scopeValidator := fetcher.NewDomainScopeValidator()

	// Initialize robots.txt service with 24 hour cache TTL
	robotsTxtService := cache.NewCachedRobotsTxtService(a.redisClient, 24*time.Hour, a.zapLogger)

	// Get queue/topic name from configuration
	parsingQueue := a.getQueueName(config.ParsingQueueKey)

	// Get tracer if telemetry is available
	var tracer trace.Tracer
	if a.telemetryProvider != nil {
		tracer = a.telemetryProvider.Tracer("parser-worker")
	}

	// Create parser worker
	a.parserWorker = worker.NewParserWorker(
		a.msgClient,
		parsingQueue, // Consume from parsing_queue
		contentStore,
		taskRepo,
		jobRepo,
		jobConfigRepo,
		outboxRepo,
		txManager,
		scopeValidator,
		robotsTxtService,
		a.zapLogger,
		tracer,
		a.metrics,
		a.workerID,
	)
	a.activeTasksCounter = a.parserWorker

	return nil
}

func (a *WorkerApp) initExportWorker() error {
	// Initialize MinIO
	minioCfg, err := env.NewMinIOConfig()
	if err != nil {
		a.zapLogger.Fatal("Failed to create MinIO config", zap.Error(err))
	}

	contentStore, err := contentstore.NewMinIOStore(
		minioCfg.Endpoint(),
		minioCfg.AccessKeyID(),
		minioCfg.SecretAccessKey(),
		minioCfg.UseSSL(),
		minioCfg.BucketName(),
		a.zapLogger,
	)
	if err != nil {
		a.zapLogger.Fatal("Failed to create MinIO store", zap.Error(err))
	}

	// Initialize repositories
	jobRepo := repos.NewCrawlRepository(a.pgClient)
	taskRepo := repos.NewCrawlTaskRepository(a.pgClient)
	txManager := transaction.NewTransactorManager(a.pgClient.DB())

	// Get tracer if telemetry is available
	var tracer trace.Tracer
	if a.telemetryProvider != nil {
		tracer = a.telemetryProvider.Tracer("export-worker")
	}

	// Create export worker with poll interval and batch size
	pollInterval := 5 * time.Second // Poll every 30 seconds
	batchSize := 10                 // Process up to 10 jobs per batch

	a.exportWorker = worker.NewExportWorker(
		jobRepo,
		taskRepo,
		contentStore,
		txManager,
		pollInterval,
		batchSize,
		a.zapLogger,
		tracer,
	)
	a.activeTasksCounter = a.exportWorker

	return nil
}

func (a *WorkerApp) initScheduleWorker() error {
	jobRepo := repos.NewCrawlRepository(a.pgClient)
	jobConfigRepo := repos.NewCrawlJobConfigRepository(a.pgClient)
	taskRepo := repos.NewCrawlTaskRepository(a.pgClient)
	outboxRepo := repos.NewOutboxRepository(a.pgClient)
	txManager := transaction.NewTransactorManager(a.pgClient.DB())

	a.scheduleWorker = worker.NewScheduleWorker(
		jobRepo,
		jobConfigRepo,
		taskRepo,
		outboxRepo,
		txManager,
		a.zapLogger,
		a.metrics,
	)
	a.activeTasksCounter = a.scheduleWorker

	return nil
}

func (a *WorkerApp) runWorker() error {
	switch a.workerType {
	case FetchWorkerType:
		a.zapLogger.Info("Fetch worker started")
		return a.fetchWorker.Start(a.consumeCtx)
	case ParserWorkerType:
		a.zapLogger.Info("Parser worker started")
		return a.parserWorker.Start(a.consumeCtx)
	case ExportWorkerType:
		a.zapLogger.Info("Export worker started")
		return a.exportWorker.Start(a.consumeCtx)
	case SchedulerWorkerType:
		a.zapLogger.Info("Scheduler worker started")
		return a.scheduleWorker.Start(a.consumeCtx)
	}
	return nil
}

// Stop gracefully stops the worker
func (a *WorkerApp) Stop() {
	if a.workerCancel != nil {
		a.zapLogger.Info("Stopping worker...")
		a.workerCancel()
	}
}

func (a *WorkerApp) initMonitor() {
	if a.grpcConfig == nil {
		return
	}

	a.workerID = os.Getenv("WORKER_ID")
	if a.workerID == "" {
		id, err := worker.NewWorkerID(string(a.workerType))
		if err != nil {
			a.zapLogger.Warn("Failed to generate worker ID", zap.Error(err))
		} else {
			a.workerID = id
		}
	}

	if a.workerID == "" {
		a.workerID = string(a.workerType)
	}

	a.monitor = worker.NewWorkerMonitor(
		a.grpcConfig.Address(),
		a.workerID,
		string(a.workerType),
		a.startedAt,
		a.currentStatus,
		a.beginDrain,
		a.forceKill,
		a.zapLogger,
	)
}

func (a *WorkerApp) activeTasksCount() int32 {
	if a.activeTasksCounter == nil {
		return 0
	}
	return a.activeTasksCounter.ActiveTasks()
}

func (a *WorkerApp) currentStatus() crawlergrpc.WorkerStatus {
	return crawlergrpc.WorkerStatus(a.status.Load())
}

func (a *WorkerApp) beginDrain() {
	a.drainOnce.Do(func() {
		a.zapLogger.Warn("Drain requested, stopping intake of new tasks")
		a.status.Store(int32(crawlergrpc.WorkerStatus_WORKER_STATUS_DRAINING))

		switch a.workerType {
		case FetchWorkerType, ParserWorkerType:
			if a.consumeCancel != nil {
				a.consumeCancel()
			}
		case ExportWorkerType:
			if a.exportWorker != nil {
				a.exportWorker.StopAccepting()
			}
		case SchedulerWorkerType:
			if a.scheduleWorker != nil {
				a.scheduleWorker.StopAccepting()
			}
		}

		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()

			for {
				if a.activeTasksCount() == 0 {
					a.zapLogger.Info("Drain complete, shutting down worker")
					a.Stop()
					return
				}

				select {
				case <-a.workerCtx.Done():
					return
				case <-ticker.C:
				}
			}
		}()
	})
}

func (a *WorkerApp) forceKill() {
	a.status.Store(int32(crawlergrpc.WorkerStatus_WORKER_STATUS_DEAD))
	a.zapLogger.Warn("Force kill requested, exiting immediately")
	os.Exit(1)
}
