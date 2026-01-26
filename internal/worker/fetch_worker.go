package worker

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	crawljobconfig "distributed-crawler/internal/domain/crawl/repos/crawl_job_config"
	crawltask "distributed-crawler/internal/domain/crawl/repos/crawl_task"
	"distributed-crawler/internal/domain/crawl/services"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/messaging/rabbitmq"
	"distributed-crawler/internal/telemetry"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// FetchWorker consumes crawl tasks, fetches pages, stores in MinIO, and publishes to parsing queue
type FetchWorker struct {
	rmqClient        rabbitmq.Client
	crawlQueue       string
	parsingQueue     string
	contentStore     services.ContentStore
	taskRepo         crawltask.CrawlTaskRepository
	jobConfigRepo    crawljobconfig.CrawlJobConfigRepository
	fetcherFactory   services.FetcherFactory
	scopeValidator   services.ScopeValidator
	rateLimiter      services.RateLimiter
	robotsTxtService services.RobotsTxtService
	logger           *zap.Logger
	activeTasks      atomic.Int64

	tracer   trace.Tracer
	metrics  *telemetry.Metrics
	workerID string
}

// NewFetchWorker creates a new fetch worker
func NewFetchWorker(
	rmqClient rabbitmq.Client,
	crawlQueue string,
	parsingQueue string,
	contentStore services.ContentStore,
	taskRepo crawltask.CrawlTaskRepository,
	jobConfigRepo crawljobconfig.CrawlJobConfigRepository,
	fetcherFactory services.FetcherFactory,
	scopeValidator services.ScopeValidator,
	rateLimiter services.RateLimiter,
	robotsTxtService services.RobotsTxtService,
	logger *zap.Logger,
	tracer trace.Tracer,
	metrics *telemetry.Metrics,
	workerID string,
) *FetchWorker {
	return &FetchWorker{
		rmqClient:        rmqClient,
		crawlQueue:       crawlQueue,
		parsingQueue:     parsingQueue,
		contentStore:     contentStore,
		taskRepo:         taskRepo,
		jobConfigRepo:    jobConfigRepo,
		fetcherFactory:   fetcherFactory,
		scopeValidator:   scopeValidator,
		rateLimiter:      rateLimiter,
		robotsTxtService: robotsTxtService,
		logger:           logger,
		tracer:           tracer,
		metrics:          metrics,
		workerID:         workerID,
	}
}

// Start starts consuming messages from crawl_queue
func (w *FetchWorker) Start(ctx context.Context) error {
	w.logger.Info("Starting fetch worker", zap.String("queue", w.crawlQueue))
	return w.rmqClient.Consume(ctx, w.crawlQueue, w.handleMessage)
}

// ActiveTasks returns the number of tasks currently being processed.
func (w *FetchWorker) ActiveTasks() int32 {
	return int32(w.activeTasks.Load())
}

// handleMessage processes a single crawl task message
func (w *FetchWorker) handleMessage(body []byte) error {
	w.activeTasks.Add(1)
	defer w.activeTasks.Add(-1)

	startTime := time.Now()

	// Parse message
	var taskMsg rabbitmq.CrawlTaskMessage
	if err := json.Unmarshal(body, &taskMsg); err != nil {
		w.logger.Error("Failed to unmarshal task message", zap.Error(err))
		return fmt.Errorf("failed to unmarshal task: %w", err)
	}

	w.logger.Info("Received crawl task",
		zap.String("task_id", taskMsg.TaskID),
		zap.String("job_id", taskMsg.JobID),
		zap.String("url", taskMsg.URL),
	)

	// Extract trace context from message and create span
	ctx := telemetry.ExtractTraceContext(context.Background(), taskMsg.TraceContext)

	var span trace.Span
	if w.tracer != nil {
		ctx, span = w.tracer.Start(ctx, "fetch_worker_process",
			trace.WithSpanKind(trace.SpanKindConsumer),
			trace.WithAttributes(
				attribute.String("job.id", taskMsg.JobID),
				attribute.String("task.id", taskMsg.TaskID),
				attribute.String("task.url", taskMsg.URL),
				attribute.String("worker.id", w.workerID),
				attribute.String("worker.type", "fetch"),
			),
		)
		defer span.End()
	}

	// Parse task ID
	taskID, err := valueobjects.NewCrawlTaskID(taskMsg.TaskID)
	if err != nil {
		w.logger.Error("Invalid task ID", zap.Error(err))
		return fmt.Errorf("invalid task ID: %w", err)
	}

	// Get existing task from database (includes Job)
	task, err := w.taskRepo.Get(ctx, taskID)
	if err != nil {
		w.logger.Error("Failed to get task",
			zap.String("task_id", taskMsg.TaskID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Load job config
	config, err := w.jobConfigRepo.Get(ctx, task.Job.JobConfigID)
	if err != nil {
		w.logger.Error("Failed to get job config",
			zap.String("task_id", taskMsg.TaskID),
			zap.String("config_id", task.Job.JobConfigID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to get job config: %w", err)
	}

	// Validate scope
	if err := w.scopeValidator.Validate(task.URL, task.Depth, config.Scopes); err != nil {
		w.logger.Warn("Task failed scope validation",
			zap.String("task_id", taskMsg.TaskID),
			zap.String("url", task.URL),
			zap.Uint64("depth", task.Depth),
			zap.Error(err),
		)

		// Mark task as failed with error message
		task.Status = models.TaskStatusFailed
		errMsg := fmt.Sprintf("scope validation failed: %v", err)
		task.ErrorMessage = &errMsg
		if updateErr := w.taskRepo.Update(ctx, *task); updateErr != nil {
			w.logger.Error("Failed to update task status to failed",
				zap.String("task_id", taskMsg.TaskID),
				zap.Error(updateErr),
			)
			return fmt.Errorf("failed to update task status: %w", updateErr)
		}

		w.logger.Info("Task marked as failed due to scope violation",
			zap.String("task_id", taskMsg.TaskID),
			zap.String("url", task.URL),
		)

		// Don't return error - task is processed (marked as failed)
		return nil
	}

	// Check robots.txt rules
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	allowed, err := w.robotsTxtService.IsAllowed(ctx, task.URL, userAgent)
	if err != nil {
		w.logger.Warn("Failed to check robots.txt, proceeding with fetch",
			zap.String("task_id", taskMsg.TaskID),
			zap.String("url", task.URL),
			zap.Error(err),
		)
		// On error, proceed with fetching (permissive default)
	} else if !allowed {
		w.logger.Warn("URL disallowed by robots.txt",
			zap.String("task_id", taskMsg.TaskID),
			zap.String("url", task.URL),
		)

		// Mark task as failed due to robots.txt with error message
		task.Status = models.TaskStatusFailed
		errMsg := "URL disallowed by robots.txt"
		task.ErrorMessage = &errMsg
		if updateErr := w.taskRepo.Update(ctx, *task); updateErr != nil {
			w.logger.Error("Failed to update task status to failed",
				zap.String("task_id", taskMsg.TaskID),
				zap.Error(updateErr),
			)
			return fmt.Errorf("failed to update task status: %w", updateErr)
		}

		w.logger.Info("Task marked as failed due to robots.txt disallow",
			zap.String("task_id", taskMsg.TaskID),
			zap.String("url", task.URL),
		)

		// Don't return error - task is processed (marked as failed)
		return nil
	}

	// Apply rate limiting before fetching
	parsedURL, err := url.Parse(task.URL)
	if err != nil || parsedURL.Host == "" {
		invalidErr := err
		if invalidErr == nil {
			invalidErr = fmt.Errorf("missing host")
		}
		w.logger.Error("Invalid URL for rate limiting",
			zap.String("task_id", taskMsg.TaskID),
			zap.String("url", task.URL),
			zap.Error(invalidErr),
		)
		return fmt.Errorf("invalid url for rate limiting: %w", invalidErr)
	}

	host := parsedURL.Hostname()
	if host == "" {
		host = parsedURL.Host
	}

	allowed, retryAfter, err := w.rateLimiter.Allow(ctx, *config, "domain", host)
	if err != nil {
		w.logger.Error("Failed to check rate limit",
			zap.String("task_id", taskMsg.TaskID),
			zap.String("url", task.URL),
			zap.String("domain", host),
			zap.Error(err),
		)
		return fmt.Errorf("failed to check rate limit: %w", err)
	}

	if !allowed {
		w.logger.Warn("Request rate limited, need to wait",
			zap.String("task_id", taskMsg.TaskID),
			zap.String("url", task.URL),
			zap.String("domain", host),
			zap.Duration("retry_after", retryAfter),
		)

		// Wait for the retry period
		time.Sleep(retryAfter)

		// Retry rate limit check after waiting
		allowed, retryAfter, err = w.rateLimiter.Allow(ctx, *config, "domain", host)
		if err != nil {
			w.logger.Error("Failed to check rate limit after retry",
				zap.String("task_id", taskMsg.TaskID),
				zap.String("domain", host),
				zap.Error(err),
			)
			return fmt.Errorf("failed to check rate limit after retry: %w", err)
		}

		if !allowed {
			// Still rate limited - this shouldn't happen after waiting, but handle it
			w.logger.Error("Still rate limited after waiting",
				zap.String("task_id", taskMsg.TaskID),
				zap.Duration("retry_after", retryAfter),
			)
			return fmt.Errorf("still rate limited after waiting %v", retryAfter)
		}
	}

	// Create configured fetcher
	fetcherInst := w.fetcherFactory.CreateFetcher(config.Auth, config.Retries)

	// Start child span for HTTP fetch
	var fetchSpan trace.Span
	if w.tracer != nil {
		ctx, fetchSpan = w.tracer.Start(ctx, "http_fetch",
			trace.WithAttributes(
				attribute.String("task.url", task.URL),
			),
		)
	}

	// Fetch the page
	fetchStart := time.Now()
	fetchResult, err := fetcherInst.Fetch(ctx, task.URL)
	fetchDuration := time.Since(fetchStart).Seconds()

	// Extract domain for metrics
	domain := extractDomain(task.URL)

	if err != nil {
		if fetchSpan != nil {
			fetchSpan.RecordError(err)
			fetchSpan.SetStatus(codes.Error, err.Error())
			fetchSpan.End()
		}

		// Record fetch error metric
		w.recordFetchError(ctx, domain, categorizeError(err))

		w.logger.Error("Failed to fetch page",
			zap.String("task_id", taskMsg.TaskID),
			zap.String("url", task.URL),
			zap.Error(err),
		)

		// Check if this is a permanent error (non-retryable)
		// These errors contain "permanent error" in the message from the fetcher
		isPermanent := strings.Contains(err.Error(), "permanent error")

		// Check if fetcher exhausted all retries
		// These errors contain "failed after X attempts" indicating retries were exhausted
		retriesExhausted := strings.Contains(err.Error(), "failed after") && strings.Contains(err.Error(), "attempts")

		// Mark task as failed for both permanent errors and exhausted retries
		if isPermanent || retriesExhausted {
			if span != nil {
				span.SetStatus(codes.Error, "fetch failed")
			}

			task.Status = models.TaskStatusFailed
			errMsg := fmt.Sprintf("fetch failed: %v", err)
			task.ErrorMessage = &errMsg
			if updateErr := w.taskRepo.Update(ctx, *task); updateErr != nil {
				w.logger.Error("Failed to update task status to failed",
					zap.String("task_id", taskMsg.TaskID),
					zap.Error(updateErr),
				)
				return fmt.Errorf("failed to update task status: %w", updateErr)
			}

			var reason string
			if isPermanent {
				reason = "permanent error"
			} else {
				reason = "retries exhausted"
			}

			w.logger.Info("Task marked as failed",
				zap.String("task_id", taskMsg.TaskID),
				zap.String("url", task.URL),
				zap.String("reason", reason),
				zap.Error(err),
			)

			// Return nil to acknowledge message and prevent requeuing
			return nil
		}

		// For transient errors that haven't exhausted retries, return error to trigger requeue/retry
		return fmt.Errorf("failed to fetch page: %w", err)
	}

	// End fetch span on success
	if fetchSpan != nil {
		fetchSpan.SetAttributes(
			attribute.Int("http.status_code", fetchResult.StatusCode),
			attribute.Int("page.size_bytes", len(fetchResult.Body)),
		)
		fetchSpan.SetStatus(codes.Ok, "")
		fetchSpan.End()
	}

	// Record successful fetch duration metric
	w.recordFetchDuration(ctx, domain, fetchResult.StatusCode, fetchDuration)

	// Check for duplicate content (deduplication)
	isDuplicate, err := w.taskRepo.ExistsByJobIDAndHashExcluding(ctx, task.JobID, fetchResult.BodyHash, task.ID)
	if err != nil {
		w.logger.Error("Failed to check for duplicate content",
			zap.String("task_id", taskMsg.TaskID),
			zap.String("body_hash", fetchResult.BodyHash),
			zap.Error(err),
		)
		return fmt.Errorf("failed to check for duplicate content: %w", err)
	}

	if isDuplicate {
		w.logger.Info("Duplicate content detected, skipping task",
			zap.String("task_id", taskMsg.TaskID),
			zap.String("url", task.URL),
			zap.String("body_hash", fetchResult.BodyHash),
		)

		// Mark task as skipped due to duplicate content
		task.Status = models.TaskStatusSkipped
		errMsg := fmt.Sprintf("duplicate content (body_hash: %s)", fetchResult.BodyHash)
		task.ErrorMessage = &errMsg
		if updateErr := w.taskRepo.Update(ctx, *task); updateErr != nil {
			w.logger.Error("Failed to update task status to skipped",
				zap.String("task_id", taskMsg.TaskID),
				zap.Error(updateErr),
			)
			return fmt.Errorf("failed to update task status: %w", updateErr)
		}

		// Task processed successfully (marked as skipped)
		return nil
	}

	// Generate MinIO object key
	minioKey := fmt.Sprintf("pages/%s/%s.html", task.JobID.String(), taskID.String())

	// Upload to MinIO
	if err := w.contentStore.Store(ctx, minioKey, fetchResult.Body, fetchResult.ContentType); err != nil {
		w.logger.Error("Failed to store content to MinIO",
			zap.String("task_id", taskMsg.TaskID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to store content to MinIO: %w", err)
	}

	// Update task with fetch results
	task.MarkAsParsed(fetchResult.FinalURL, fetchResult.BodyHash, minioKey)

	// Save task to database
	if err := w.taskRepo.Update(ctx, *task); err != nil {
		w.logger.Error("Failed to update task",
			zap.String("task_id", taskMsg.TaskID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Publish to parsing queue only after successful DB save
	parsingMsg := rabbitmq.ParsingTaskMessage{
		TaskID:       taskMsg.TaskID,
		JobID:        taskMsg.JobID,
		EnqueuedAt:   time.Now(),
		TraceContext: telemetry.InjectTraceContext(ctx),
	}

	if err := w.rmqClient.Publish(ctx, w.parsingQueue, parsingMsg); err != nil {
		w.logger.Error("Failed to publish to parsing queue",
			zap.String("task_id", taskMsg.TaskID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to publish to parsing queue: %w", err)
	}

	// Set span attributes for successful completion
	if span != nil {
		span.SetAttributes(
			attribute.Int("page.size_bytes", len(fetchResult.Body)),
			attribute.String("http.final_url", fetchResult.FinalURL),
		)
		span.SetStatus(codes.Ok, "")
	}

	w.logger.Info("Successfully processed fetch task",
		zap.String("task_id", taskMsg.TaskID),
		zap.String("url", task.URL),
		zap.String("final_url", fetchResult.FinalURL),
		zap.String("body_hash", fetchResult.BodyHash),
		zap.String("minio_key", minioKey),
		zap.Duration("total_duration", time.Since(startTime)),
	)

	return nil
}

// recordFetchDuration records the fetch duration metric
func (w *FetchWorker) recordFetchDuration(ctx context.Context, domain string, statusCode int, duration float64) {
	if w.metrics == nil || w.metrics.FetchDuration == nil {
		return
	}
	w.metrics.FetchDuration.Record(ctx, duration,
		metric.WithAttributes(
			attribute.String("domain", domain),
			attribute.Int("status_code", statusCode),
		),
	)
}

// recordFetchError records the fetch error metric
func (w *FetchWorker) recordFetchError(ctx context.Context, domain, errorType string) {
	if w.metrics == nil || w.metrics.FetchErrorsTotal == nil {
		return
	}
	w.metrics.FetchErrorsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("domain", domain),
			attribute.String("error_type", errorType),
		),
	)
}

// extractDomain extracts the domain from a URL for metric labels
func extractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "unknown"
	}
	host := parsed.Hostname()
	if host == "" {
		return "unknown"
	}
	return host
}

// categorizeError categorizes an error for metric labels
func categorizeError(err error) string {
	errStr := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errStr, "timeout"):
		return "timeout"
	case strings.Contains(errStr, "connection refused"):
		return "connection_refused"
	case strings.Contains(errStr, "no such host"):
		return "dns_error"
	case strings.Contains(errStr, "permanent error"):
		return "permanent"
	case strings.Contains(errStr, "403"), strings.Contains(errStr, "forbidden"):
		return "http_403"
	case strings.Contains(errStr, "404"), strings.Contains(errStr, "not found"):
		return "http_404"
	case strings.Contains(errStr, "5"):
		return "http_5xx"
	default:
		return "other"
	}
}
