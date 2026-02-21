package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"distributed-crawler/internal/domain/crawl/events"
	"distributed-crawler/internal/domain/crawl/models"
	crawljob "distributed-crawler/internal/domain/crawl/repos/crawl_job"
	crawljobconfig "distributed-crawler/internal/domain/crawl/repos/crawl_job_config"
	crawltask "distributed-crawler/internal/domain/crawl/repos/crawl_task"
	"distributed-crawler/internal/domain/crawl/repos/outbox"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/telemetry"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

const (
	DefaultSchedulePollInterval = 10 * time.Second
	DefaultScheduleBatchSize    = 100
)

// ScheduleWorker creates crawl jobs based on cron schedules.
type ScheduleWorker struct {
	jobRepo       crawljob.CrawlJobRepository
	jobConfigRepo crawljobconfig.CrawlJobConfigRepository
	taskRepo      crawltask.CrawlTaskRepository
	outboxRepo    outbox.OutboxRepository
	txManager     persistence.TxManager
	pollInterval  time.Duration
	batchSize     int
	logger        *zap.Logger
	metrics       *telemetry.Metrics
	activeTasks   atomic.Int64
	accepting     atomic.Bool
}

// NewScheduleWorker creates a new schedule worker.
func NewScheduleWorker(
	jobRepo crawljob.CrawlJobRepository,
	jobConfigRepo crawljobconfig.CrawlJobConfigRepository,
	taskRepo crawltask.CrawlTaskRepository,
	outboxRepo outbox.OutboxRepository,
	txManager persistence.TxManager,
	logger *zap.Logger,
	metrics *telemetry.Metrics,
) *ScheduleWorker {
	worker := &ScheduleWorker{
		jobRepo:       jobRepo,
		jobConfigRepo: jobConfigRepo,
		taskRepo:      taskRepo,
		outboxRepo:    outboxRepo,
		txManager:     txManager,
		pollInterval:  DefaultSchedulePollInterval,
		batchSize:     DefaultScheduleBatchSize,
		logger:        logger,
		metrics:       metrics,
	}
	worker.accepting.Store(true)
	return worker
}

// Start starts the scheduling loop.
func (w *ScheduleWorker) Start(ctx context.Context) error {
	w.logger.Info("Starting schedule worker",
		zap.Duration("poll_interval", w.pollInterval),
		zap.Int("batch_size", w.batchSize),
	)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	w.processSchedules(ctx)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Schedule worker stopped")
			return ctx.Err()
		case <-ticker.C:
			if !w.accepting.Load() {
				w.logger.Info("Schedule worker drain completed")
				return nil
			}
			w.processSchedules(ctx)
		}
	}
}

// ActiveTasks returns the number of schedule configs currently being processed.
func (w *ScheduleWorker) ActiveTasks() int32 {
	return int32(w.activeTasks.Load())
}

// StopAccepting prevents new scheduling cycles from starting.
func (w *ScheduleWorker) StopAccepting() {
	w.accepting.Store(false)
}

func (w *ScheduleWorker) processSchedules(ctx context.Context) {
	offset := 0

	for {
		configs, err := w.jobConfigRepo.ListAllScheduled(ctx, w.batchSize, offset)
		if err != nil {
			w.logger.Error("Failed to list crawl job configs for scheduling", zap.Error(err))
			return
		}

		if len(configs) == 0 {
			return
		}

		for _, config := range configs {
			if err := w.processConfig(ctx, config); err != nil {
				w.logger.Error("Failed to process schedule config",
					zap.String("config_id", config.ID.String()),
					zap.Error(err),
				)
			}
		}

		if len(configs) < w.batchSize {
			return
		}
		offset += len(configs)
	}
}

func (w *ScheduleWorker) processConfig(ctx context.Context, config *models.CrawlJobConfig) error {
	w.activeTasks.Add(1)
	defer w.activeTasks.Add(-1)

	cronExpr := strings.TrimSpace(config.Schedule.Cron)
	if cronExpr == "" {
		return nil
	}

	schedule, err := cron.ParseStandard(cronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", cronExpr, err)
	}

	now := time.Now().UTC()
	if config.Schedule.NextRunAt == nil {
		lastJob, err := w.jobRepo.GetLatestByConfigID(ctx, config.ID)
		if err != nil {
			return fmt.Errorf("failed to load last job for config %s: %w", config.ID.String(), err)
		}
		if lastJob != nil && lastJob.CompletedAt != nil {
			lastRun := lastJob.CreatedAt.UTC()
			config.Schedule.LastRunAt = &lastRun
		}

		baseTime := now
		if config.Schedule.LastRunAt != nil {
			baseTime = config.Schedule.LastRunAt.UTC()
		}

		nextRun := schedule.Next(baseTime)
		config.Schedule.NextRunAt = &nextRun
		return w.jobConfigRepo.Update(ctx, *config)
	}

	if config.Schedule.NextRunAt.After(now) {
		return nil
	}

	lastJob, err := w.jobRepo.GetLatestByConfigID(ctx, config.ID)
	if err != nil {
		return fmt.Errorf("failed to load last job for config %s: %w", config.ID.String(), err)
	}

	if lastJob != nil && lastJob.CompletedAt == nil {
		return nil
	}

	return w.createScheduledJob(ctx, config, schedule, now)
}

func (w *ScheduleWorker) createScheduledJob(
	ctx context.Context,
	config *models.CrawlJobConfig,
	schedule cron.Schedule,
	now time.Time,
) error {
	return w.txManager.ReadCommitted(ctx, func(ctxTX context.Context) error {
		nextRun := schedule.Next(now)
		config.Schedule.LastRunAt = &now
		config.Schedule.NextRunAt = &nextRun

		if err := w.jobConfigRepo.Update(ctxTX, *config); err != nil {
			return fmt.Errorf("failed to update schedule config: %w", err)
		}

		jobID := valueobjects.GenerateCrawlJobID()
		crawlJob := models.CrawlJob{
			ID:           jobID,
			JobConfigID:  config.ID,
			JobConfig:    config,
			Status:       models.TaskStatusInProgress,
			CreatedAt:    now,
			ExportStatus: models.ExportStatusNotStarted,
		}

		if _, err := w.jobRepo.Create(ctxTX, crawlJob); err != nil {
			return fmt.Errorf("failed to create scheduled crawl job: %w", err)
		}

		tasks := make([]models.CrawlTask, 0, len(config.Seeds))
		outboxEvents := make([]models.OutboxEvent, 0, len(config.Seeds))

		for _, seed := range config.Seeds {
			task := models.CrawlTask{
				ID:         valueobjects.GenerateCrawlTaskID(),
				JobID:      jobID,
				URL:        seed.Url,
				Status:     models.TaskStatusInProgress,
				EnqueuedAt: now,
				Depth:      0,
			}
			tasks = append(tasks, task)

			event := events.NewTaskEnqueuedEvent(
				task.ID.String(),
				task.JobID.String(),
				task.URL,
				task.EnqueuedAt,
			)

			payload, err := json.Marshal(event)
			if err != nil {
				return fmt.Errorf("failed to marshal task enqueue event: %w", err)
			}

			outboxEvents = append(outboxEvents, models.OutboxEvent{
				ID:          valueobjects.GenerateOutboxEventID(),
				EventType:   string(event.Type),
				AggregateID: task.ID.String(),
				Payload:     payload,
				OccurredAt:  event.OccurredAt,
				ProcessedAt: nil,
				CreatedAt:   now,
			})
		}

		if _, err := w.taskRepo.BulkCreate(ctxTX, tasks); err != nil {
			return fmt.Errorf("failed to create scheduled crawl tasks: %w", err)
		}

		if err := w.outboxRepo.BulkCreate(ctxTX, outboxEvents); err != nil {
			return fmt.Errorf("failed to create scheduled outbox events: %w", err)
		}

		if w.metrics != nil && w.metrics.TasksCreatedTotal != nil {
			w.metrics.TasksCreatedTotal.Add(ctxTX, int64(len(tasks)))
		}

		w.logger.Info("Scheduled new crawl job",
			zap.String("job_id", jobID.String()),
			zap.String("config_id", config.ID.String()),
			zap.Time("next_run_at", nextRun),
		)

		return nil
	})
}
