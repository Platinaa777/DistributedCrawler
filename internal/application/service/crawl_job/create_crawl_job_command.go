package crawljob

import (
	"context"
	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/events"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"encoding/json"
	"fmt"
	"time"
)

func (s *crawlJobServ) CreateCrawlJob(ctx context.Context, command service.CreateCrawlJobCommand) (valueobjects.CrawlJobID, error) {
	if len(command.Config.Seeds) == 0 {
		return valueobjects.CrawlJobID{}, fmt.Errorf("seeds list cannot be empty")
	}

	var jobID valueobjects.CrawlJobID

	err := s.txManager.ReadCommitted(ctx, func(ctx context.Context) error {
		// Generate IDs
		configID := valueobjects.GenerateID()
		jobID = valueobjects.GenerateCrawlJobID()

		// Set config ID
		config := command.Config
		config.ID = configID

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
			Status:       models.TaskStatusInProgress,
			CreatedAt:    time.Now(),
			ExportStatus: models.ExportStatusNotStarted,
		}

		id, err := s.crawlJobRepo.Create(ctx, crawlJob)
		if err != nil {
			return fmt.Errorf("failed to create crawl job: %w", err)
		}
		jobID = id

		// Create crawl tasks and outbox events
		tasks := make([]models.CrawlTask, 0, len(config.Seeds))
		now := time.Now()

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
				CreatedAt:   time.Now(),
			}

			if err := s.outboxRepo.Create(ctx, outboxEvent); err != nil {
				return fmt.Errorf("failed to create outbox event: %w", err)
			}
		}

		if err := s.crawlTaskRepo.BulkCreate(ctx, tasks); err != nil {
			return fmt.Errorf("failed to create crawl tasks: %w", err)
		}

		return nil
	})

	if err != nil {
		return valueobjects.CrawlJobID{}, err
	}

	return jobID, nil
}
