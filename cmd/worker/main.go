package main

import (
	"context"
	"distributed-crawler/internal/config"
	"distributed-crawler/internal/config/env"
	"distributed-crawler/internal/infra/logger"
	"distributed-crawler/internal/infra/messaging/rabbitmq"
	"distributed-crawler/internal/worker"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "config-path", ".env", "path to config file")
}

func main() {
	flag.Parse()

	// Load configuration
	if err := config.Load(configPath); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	loggerConfig, err := env.NewLoggerConfig()
	if err != nil {
		log.Fatalf("Failed to get logger config: %v", err)
	}

	if err := logger.InitWithConfig(loggerConfig.Level(), loggerConfig.Env()); err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer logger.Sync()

	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create zap logger: %v", err)
	}

	// Load RabbitMQ config
	rmqConfig, err := env.NewRabbitMQConfig()
	if err != nil {
		log.Fatalf("Failed to get RabbitMQ config: %v", err)
	}

	// Create RabbitMQ client
	rmqClient, err := rabbitmq.NewClient(rmqConfig.URL())
	if err != nil {
		log.Fatalf("Failed to create RabbitMQ client: %v", err)
	}
	defer func() {
		if err := rmqClient.Close(); err != nil {
			zapLogger.Error("Failed to close RabbitMQ client", zap.Error(err))
		}
	}()

	// Create scraper worker
	scraperWorker := worker.NewScraperWorker(
		rmqClient,
		rmqConfig.QueueName(),
		zapLogger,
	)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start worker in goroutine
	errChan := make(chan error, 1)
	go func() {
		zapLogger.Info("Starting scraper worker",
			zap.String("queue", rmqConfig.QueueName()),
			zap.String("rabbitmq_url", rmqConfig.URL()),
		)
		if err := scraperWorker.Start(ctx); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		zapLogger.Info("Received shutdown signal, stopping worker...")
		cancel()
	case err := <-errChan:
		zapLogger.Error("Worker error", zap.Error(err))
		cancel()
	}

	zapLogger.Info("Scraper worker stopped")
}