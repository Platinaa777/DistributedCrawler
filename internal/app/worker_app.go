package app

import (
	"context"
	"flag"
	"log"
	"sync"

	"distributed-crawler/internal/config"
	"distributed-crawler/internal/config/env"
	"distributed-crawler/internal/infra/messaging/rabbitmq"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/infra/persistence/postgres/pg"
	"distributed-crawler/internal/infra/persistence/postgres/repos"
	"distributed-crawler/internal/infra/services/contentstore"
	"distributed-crawler/internal/worker"

	"go.uber.org/zap"
)

var workerConfigPath string

func init() {
	flag.StringVar(&workerConfigPath, "worker-config-path", ".worker.env", "path to config file")
}

// WorkerType defines the type of worker to run
type WorkerType string

const (
	FetchWorkerType  WorkerType = "fetch"
	ParserWorkerType WorkerType = "parser"
)

type WorkerApp struct {
	workerType   WorkerType
	zapLogger    *zap.Logger
	pgClient     persistence.Client
	rmqClient    rabbitmq.Client
	rmqConfig    config.RabbitMQConfig

	fetchWorker  *worker.FetchWorker
	parserWorker *worker.ParserWorker

	workerCtx    context.Context
	workerCancel context.CancelFunc
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
		if a.pgClient != nil {
			a.pgClient.Close()
		}
		if a.rmqClient != nil {
			a.rmqClient.Close()
		}
		if a.zapLogger != nil {
			a.zapLogger.Sync()
		}
	}()

	wg := sync.WaitGroup{}

	wg.Go(func() {
		a.runWorker()
	})

	wg.Wait()

	return nil
}

func (a *WorkerApp) initDeps(ctx context.Context) error {
	inits := []func(context.Context) error{
		a.initConfig,
		a.initLogger,
		a.initPostgreSQL,
		a.initRabbitMQ,
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

	a.zapLogger = zapLogger
	return nil
}

func (a *WorkerApp) initPostgreSQL(ctx context.Context) error {
	pgCfg, err := env.NewPGConfig()
	if err != nil {
		a.zapLogger.Fatal("Failed to create PG config", zap.Error(err))
	}

	pgClient, err := pg.New(ctx, pgCfg.DSN())
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

func (a *WorkerApp) initRabbitMQ(_ context.Context) error {
	rmqCfg, err := env.NewRabbitMQConfig()
	if err != nil {
		a.zapLogger.Fatal("Failed to create RabbitMQ config", zap.Error(err))
	}

	rmqClient, err := rabbitmq.NewClient(rmqCfg.URL())
	if err != nil {
		a.zapLogger.Fatal("Failed to create RabbitMQ client", zap.Error(err))
	}

	a.rmqClient = rmqClient
	a.rmqConfig = rmqCfg
	return nil
}

func (a *WorkerApp) initWorker(ctx context.Context) error {
	a.workerCtx, a.workerCancel = context.WithCancel(ctx)

	switch a.workerType {
	case FetchWorkerType:
		return a.initFetchWorker()
	case ParserWorkerType:
		return a.initParserWorker()
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

	// Get queue names from configuration
	crawlQueue := a.rmqConfig.GetQueueName(config.CrawlQueueKey)
	parsingQueue := a.rmqConfig.GetQueueName(config.ParsingQueueKey)

	// Create fetch worker
	a.fetchWorker = worker.NewFetchWorker(
		a.rmqClient,
		crawlQueue,   // Consume from crawl_queue
		parsingQueue, // Publish to parsing_queue
		contentStore,
		taskRepo,
		a.zapLogger,
	)

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

	// Get queue name from configuration
	parsingQueue := a.rmqConfig.GetQueueName(config.ParsingQueueKey)

	// Create parser worker
	a.parserWorker = worker.NewParserWorker(
		a.rmqClient,
		parsingQueue, // Consume from parsing_queue
		contentStore,
		taskRepo,
		jobRepo,
		jobConfigRepo,
		a.zapLogger,
	)

	return nil
}

func (a *WorkerApp) runWorker() {
	switch a.workerType {
	case FetchWorkerType:
		a.zapLogger.Info("Fetch worker started")
		if err := a.fetchWorker.Start(a.workerCtx); err != nil {
			a.zapLogger.Fatal("Fetch worker failed", zap.Error(err))
		}
	case ParserWorkerType:
		a.zapLogger.Info("Parser worker started")
		if err := a.parserWorker.Start(a.workerCtx); err != nil {
			a.zapLogger.Fatal("Parser worker failed", zap.Error(err))
		}
	}
}

// Stop gracefully stops the worker
func (a *WorkerApp) Stop() {
	if a.workerCancel != nil {
		a.zapLogger.Info("Stopping worker...")
		a.workerCancel()
	}
}
