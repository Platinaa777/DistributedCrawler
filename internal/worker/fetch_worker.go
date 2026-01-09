package worker

import (
	"context"
	"crypto/sha256"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/repos/page_fetch"
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
	fetchRepo     page_fetch.PageFetchRepository
	httpClient    *http.Client
	logger        *zap.Logger
}

// NewFetchWorker creates a new fetch worker
func NewFetchWorker(
	rmqClient rabbitmq.Client,
	crawlQueue string,
	parsingQueue string,
	contentStore services.ContentStore,
	fetchRepo page_fetch.PageFetchRepository,
	logger *zap.Logger,
) *FetchWorker {
	return &FetchWorker{
		rmqClient:    rmqClient,
		crawlQueue:   crawlQueue,
		parsingQueue: parsingQueue,
		contentStore: contentStore,
		fetchRepo:    fetchRepo,
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
	var task rabbitmq.CrawlTaskMessage
	if err := json.Unmarshal(body, &task); err != nil {
		w.logger.Error("Failed to unmarshal task message", zap.Error(err))
		return fmt.Errorf("failed to unmarshal task: %w", err)
	}

	w.logger.Info("Received crawl task",
		zap.String("task_id", task.TaskID),
		zap.String("job_id", task.JobID),
		zap.String("url", task.URL),
	)

	ctx := context.Background()

	// Fetch the page
	fetchResult, err := w.fetchPage(ctx, task)
	if err != nil {
		w.logger.Error("Failed to fetch page",
			zap.String("task_id", task.TaskID),
			zap.String("url", task.URL),
			zap.Error(err),
		)
		return fmt.Errorf("failed to fetch page: %w", err)
	}

	// Save fetch metadata to database
	if err := w.fetchRepo.Save(ctx, fetchResult.Metadata); err != nil {
		w.logger.Error("Failed to save fetch metadata",
			zap.String("task_id", task.TaskID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save fetch metadata: %w", err)
	}

	// Publish to parsing queue only after successful DB save
	parsingMsg := rabbitmq.ParsingTaskMessage{
		TaskID:     task.TaskID,
		JobID:      task.JobID,
		EnqueuedAt: time.Now(),
	}

	if err := w.rmqClient.Publish(ctx, w.parsingQueue, parsingMsg); err != nil {
		w.logger.Error("Failed to publish to parsing queue",
			zap.String("task_id", task.TaskID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to publish to parsing queue: %w", err)
	}

	w.logger.Info("Successfully processed fetch task",
		zap.String("task_id", task.TaskID),
		zap.String("url", task.URL),
		zap.Int("status_code", fetchResult.Metadata.StatusCode),
		zap.Int("duration_ms", fetchResult.Metadata.DurationMs),
		zap.String("minio_key", fetchResult.Metadata.MinioObjectKey),
	)

	return nil
}

// fetchResult contains the results of a page fetch
type fetchResult struct {
	Metadata *models.PageFetch
	Body     []byte
}

// fetchPage performs HTTP GET and collects metadata
func (w *FetchWorker) fetchPage(ctx context.Context, task rabbitmq.CrawlTaskMessage) (*fetchResult, error) {
	startTime := time.Now()

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

	duration := time.Since(startTime)

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Calculate SHA-256 hash
	hash := sha256.Sum256(body)
	bodyHash := hex.EncodeToString(hash[:])

	// Collect headers (first value only for simplicity)
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

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

	// Parse IDs
	taskID, err := valueobjects.NewCrawlTaskID(task.TaskID)
	if err != nil {
		return nil, fmt.Errorf("invalid task ID: %w", err)
	}

	jobID, err := valueobjects.NewCrawlJobID(task.JobID)
	if err != nil {
		return nil, fmt.Errorf("invalid job ID: %w", err)
	}

	// Build fetch metadata
	finalURL := resp.Request.URL.String()
	contentLength := resp.ContentLength

	metadata := &models.PageFetch{
		TaskID:         taskID,
		JobID:          jobID,
		URL:            task.URL,
		FinalURL:       &finalURL,
		StatusCode:     resp.StatusCode,
		DurationMs:     int(duration.Milliseconds()),
		Headers:        headers,
		ContentType:    &contentType,
		ContentLength:  &contentLength,
		BodyHash:       bodyHash,
		MinioObjectKey: minioKey,
		FetchedAt:      time.Now(),
		CreatedAt:      time.Now(),
	}

	return &fetchResult{
		Metadata: metadata,
		Body:     body,
	}, nil
}
