package worker

import (
	"context"
	"crypto/sha256"
	"distributed-crawler/internal/domain/crawl/repos/crawl_task"
	"distributed-crawler/internal/domain/crawl/services"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/messaging/rabbitmq"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// FetchWorker consumes crawl tasks, fetches pages, stores in MinIO, and publishes to parsing queue
type FetchWorker struct {
	rmqClient     rabbitmq.Client
	crawlQueue    string
	parsingQueue  string
	contentStore  services.ContentStore
	taskRepo      crawltask.CrawlTaskRepository
	httpClient    *http.Client
	logger        *zap.Logger
}

// NewFetchWorker creates a new fetch worker
func NewFetchWorker(
	rmqClient rabbitmq.Client,
	crawlQueue string,
	parsingQueue string,
	contentStore services.ContentStore,
	taskRepo crawltask.CrawlTaskRepository,
	logger *zap.Logger,
) *FetchWorker {
	return &FetchWorker{
		rmqClient:    rmqClient,
		crawlQueue:   crawlQueue,
		parsingQueue: parsingQueue,
		contentStore: contentStore,
		taskRepo:     taskRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Allow up to 10 redirects
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				return nil
			},
		},
		logger: logger,
	}
}

// Start starts consuming messages from crawl_queue
func (w *FetchWorker) Start(ctx context.Context) error {
	w.logger.Info("Starting fetch worker", zap.String("queue", w.crawlQueue))
	return w.rmqClient.Consume(ctx, w.crawlQueue, w.handleMessage)
}

// handleMessage processes a single crawl task message
func (w *FetchWorker) handleMessage(body []byte) error {
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

	ctx := context.Background()

	// Parse task ID
	taskID, err := valueobjects.NewCrawlTaskID(taskMsg.TaskID)
	if err != nil {
		w.logger.Error("Invalid task ID", zap.Error(err))
		return fmt.Errorf("invalid task ID: %w", err)
	}

	// Get existing task from database
	task, err := w.taskRepo.Get(ctx, taskID)
	if err != nil {
		w.logger.Error("Failed to get task",
			zap.String("task_id", taskMsg.TaskID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Fetch the page
	fetchResult, err := w.fetchPage(ctx, taskMsg)
	if err != nil {
		w.logger.Error("Failed to fetch page",
			zap.String("task_id", taskMsg.TaskID),
			zap.String("url", taskMsg.URL),
			zap.Error(err),
		)
		return fmt.Errorf("failed to fetch page: %w", err)
	}

	// Update task with fetch results
	task.BodyHash = fetchResult.BodyHash
	task.MinioObjectKey = fetchResult.MinioKey
	task.FinalURL = &fetchResult.FinalURL

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
		TaskID:     taskMsg.TaskID,
		JobID:      taskMsg.JobID,
		EnqueuedAt: time.Now(),
	}

	if err := w.rmqClient.Publish(ctx, w.parsingQueue, parsingMsg); err != nil {
		w.logger.Error("Failed to publish to parsing queue",
			zap.String("task_id", taskMsg.TaskID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to publish to parsing queue: %w", err)
	}

	w.logger.Info("Successfully processed fetch task",
		zap.String("task_id", taskMsg.TaskID),
		zap.String("url", taskMsg.URL),
		zap.String("final_url", fetchResult.FinalURL),
		zap.String("body_hash", fetchResult.BodyHash),
		zap.String("minio_key", fetchResult.MinioKey),
	)

	return nil
}

// fetchResult contains the results of a page fetch
type fetchResult struct {
	BodyHash  string
	MinioKey  string
	FinalURL  string
}

// fetchPage performs HTTP GET, stores to MinIO, and returns fetch metadata
func (w *FetchWorker) fetchPage(ctx context.Context, task rabbitmq.CrawlTaskMessage) (*fetchResult, error) {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, task.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent
	req.Header.Set("User-Agent", "DistributedCrawler/1.0")

	// Perform request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Calculate SHA-256 hash
	hash := sha256.Sum256(body)
	bodyHash := hex.EncodeToString(hash[:])

	// Generate MinIO object key
	minioKey := fmt.Sprintf("pages/%s/%s.html", task.JobID, task.TaskID)

	// Upload to MinIO
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/html"
	}

	if err := w.contentStore.Store(ctx, minioKey, body, contentType); err != nil {
		return nil, fmt.Errorf("failed to store content to MinIO: %w", err)
	}

	// Get final URL after redirects
	finalURL := resp.Request.URL.String()

	return &fetchResult{
		BodyHash: bodyHash,
		MinioKey: minioKey,
		FinalURL: finalURL,
	}, nil
}
