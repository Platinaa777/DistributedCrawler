package worker

import (
	"context"
	"time"

	"distributed-crawler/internal/domain/crawl/models"
	crawltask "distributed-crawler/internal/domain/crawl/repos/crawl_task"
	"distributed-crawler/internal/infra/messaging"
	"distributed-crawler/internal/telemetry"

	"go.uber.org/zap"
)

const (
	DefaultStuckTaskPollInterval = 5 * time.Minute
	DefaultStuckTaskThreshold    = 10 * time.Minute
	DefaultStuckTaskBatchSize    = 500
)

// StuckTaskRecovery periodically finds InProgress tasks that were never dispatched
// (e.g. parser crashed between DB commit and Publish) and re-publishes them to the crawl queue.
type StuckTaskRecovery struct {
	taskRepo        crawltask.CrawlTaskRepository
	msgClient       messaging.Client
	availableQueues []string
	logger          *zap.Logger
	pollInterval    time.Duration
	staleThreshold  time.Duration
	batchSize       int
}

func NewStuckTaskRecovery(
	taskRepo crawltask.CrawlTaskRepository,
	msgClient messaging.Client,
	availableQueues []string,
	logger *zap.Logger,
) *StuckTaskRecovery {
	return &StuckTaskRecovery{
		taskRepo:        taskRepo,
		msgClient:       msgClient,
		availableQueues: availableQueues,
		logger:          logger,
		pollInterval:    DefaultStuckTaskPollInterval,
		staleThreshold:  DefaultStuckTaskThreshold,
		batchSize:       DefaultStuckTaskBatchSize,
	}
}

// Start runs the recovery loop until ctx is cancelled.
func (r *StuckTaskRecovery) Start(ctx context.Context) {
	r.logger.Info("Starting stuck task recovery",
		zap.Duration("poll_interval", r.pollInterval),
		zap.Duration("stale_threshold", r.staleThreshold),
	)

	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.recover(ctx)
		}
	}
}

func (r *StuckTaskRecovery) recover(ctx context.Context) {
	olderThan := time.Now().UTC().Add(-r.staleThreshold)

	tasks, err := r.taskRepo.ListStaleInProgress(ctx, olderThan, r.batchSize)
	if err != nil {
		r.logger.Error("Failed to list stale in-progress tasks", zap.Error(err))
		return
	}
	if len(tasks) == 0 {
		return
	}

	r.logger.Warn("Found stuck InProgress tasks, re-publishing",
		zap.Int("count", len(tasks)),
		zap.Time("older_than", olderThan),
	)

	traceCtx := telemetry.InjectTraceContext(ctx)
	republished := 0
	for _, task := range tasks {
		targetQueue := models.SelectCrawlQueue(r.availableQueues, nil)
		msg := messaging.CrawlTaskMessage{
			TaskID:       task.ID.String(),
			JobID:        task.JobID.String(),
			URL:          task.URL,
			EnqueuedAt:   task.EnqueuedAt,
			TraceContext: traceCtx,
		}
		if err := r.msgClient.Publish(ctx, targetQueue, msg); err != nil {
			r.logger.Error("Failed to re-publish stuck task",
				zap.String("task_id", task.ID.String()),
				zap.String("url", task.URL),
				zap.Error(err),
			)
			continue
		}
		republished++
	}

	r.logger.Info("Stuck task recovery complete",
		zap.Int("republished", republished),
		zap.Int("total_stale", len(tasks)),
	)
}

