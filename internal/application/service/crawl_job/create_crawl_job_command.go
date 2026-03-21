package crawljob

import (
	"context"
	"distributed-crawler/internal/application/service"
	authvalueobjects "distributed-crawler/internal/domain/auth/valueobjects"
	"distributed-crawler/internal/domain/crawl/events"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/telemetry"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (s *crawlJobServ) CreateCrawlJob(ctx context.Context, command service.CreateCrawlJobCommand) (valueobjects.CrawlJobID, error) {
	if len(command.Config.Seeds) == 0 {
		return valueobjects.CrawlJobID{}, fmt.Errorf("seeds list cannot be empty")
	}
	normalizedAllowedPatterns := make([]string, 0, len(command.Config.Scopes.AllowedURLPatterns))
	for _, pattern := range command.Config.Scopes.AllowedURLPatterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		normalizedAllowedPatterns = append(normalizedAllowedPatterns, pattern)
	}
	command.Config.Scopes.AllowedURLPatterns = normalizedAllowedPatterns
	if _, err := models.CompileAllowedURLPatterns(command.Config.Scopes.AllowedURLPatterns); err != nil {
		return valueobjects.CrawlJobID{}, fmt.Errorf("invalid scopes.allowed_url_patterns: %w", err)
	}
	userID, err := authvalueobjects.NewUserID(command.UserID)
	if err != nil {
		return valueobjects.CrawlJobID{}, fmt.Errorf("invalid user_id: %w", err)
	}

	var jobID valueobjects.CrawlJobID

	err = s.txManager.ReadCommitted(ctx, func(ctx context.Context) error {
		// Generate IDs
		configID := valueobjects.GenerateID()
		jobID = valueobjects.GenerateCrawlJobID()

		// Set config ID
		config := command.Config
		config.ID = configID
		config.UserID = userID

		// Create job config first
		createdConfigID, err := s.crawlJobConfigRepo.Create(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to create job config: %w", err)
		}

		// Create crawl job
		crawlJob := models.CrawlJob{
			ID:           jobID,
			JobConfigID:  createdConfigID,
			JobConfig:    &config,
			UserID:       userID,
			Status:       models.TaskStatusInProgress,
			CreatedAt:    time.Now().UTC(),
			ExportStatus: models.ExportStatusNotStarted,
		}

		id, err := s.crawlJobRepo.Create(ctx, crawlJob)
		if err != nil {
			return fmt.Errorf("failed to create crawl job: %w", err)
		}
		jobID = id

		// Create crawl tasks and outbox events
		tasks := make([]models.CrawlTask, 0, len(config.Seeds))
		now := time.Now().UTC()

		for _, seed := range config.Seeds {
			task := models.CrawlTask{
				ID:         valueobjects.GenerateCrawlTaskID(),
				JobID:      jobID,
				URL:        seed.Url,
				Status:     models.TaskStatusInProgress,
				EnqueuedAt: now,
				Depth:      0, // Seeds start at depth 0
			}
			tasks = append(tasks, task)

			// Create outbox event for this task
			event := events.NewTaskEnqueuedEvent(
				task.ID.String(),
				task.JobID.String(),
				task.URL,
				task.EnqueuedAt,
			)
			// Capture the gRPC request's trace context so the outbox publisher
			// can continue the same trace when it publishes to RabbitMQ.
			event.TraceContext = telemetry.InjectTraceContext(ctx)

			// Marshal event to JSON
			payload, err := json.Marshal(event)
			if err != nil {
				return fmt.Errorf("failed to marshal event: %w", err)
			}

			// Store event in outbox
			outboxEvent := models.OutboxEvent{
				ID:          valueobjects.GenerateOutboxEventID(),
				EventType:   string(event.Type),
				AggregateID: task.ID.String(),
				Payload:     payload,
				OccurredAt:  event.OccurredAt,
				ProcessedAt: nil,
				CreatedAt:   time.Now().UTC(),
			}

			if err := s.outboxRepo.Create(ctx, outboxEvent); err != nil {
				return fmt.Errorf("failed to create outbox event: %w", err)
			}
		}

		if _, err := s.crawlTaskRepo.BulkCreate(ctx, tasks); err != nil {
			return fmt.Errorf("failed to create crawl tasks: %w", err)
		}

		// Record tasks created metric
		if s.metrics != nil && s.metrics.TasksCreatedTotal != nil {
			s.metrics.TasksCreatedTotal.Add(ctx, int64(len(tasks)))
		}

		return nil
	})

	if err != nil {
		return valueobjects.CrawlJobID{}, err
	}

	return jobID, nil
}
