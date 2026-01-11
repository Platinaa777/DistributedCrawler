package worker

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"distributed-crawler/internal/domain/crawl/events"
	"distributed-crawler/internal/domain/crawl/models"
	crawljob "distributed-crawler/internal/domain/crawl/repos/crawl_job"
	crawljobconfig "distributed-crawler/internal/domain/crawl/repos/crawl_job_config"
	crawltask "distributed-crawler/internal/domain/crawl/repos/crawl_task"
	"distributed-crawler/internal/domain/crawl/repos/outbox"
	"distributed-crawler/internal/domain/crawl/services"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/messaging/rabbitmq"
	"distributed-crawler/internal/infra/persistence"

	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"
)

// ParserWorker consumes parsing tasks, loads HTML from MinIO, parses using DSL, and prints results
type ParserWorker struct {
	rmqClient      rabbitmq.Client
	parsingQueue   string
	contentStore   services.ContentStore
	taskRepo       crawltask.CrawlTaskRepository
	jobRepo        crawljob.CrawlJobRepository
	jobConfigRepo  crawljobconfig.CrawlJobConfigRepository
	outboxRepo     outbox.OutboxRepository
	txManager      persistence.TxManager
	scopeValidator services.ScopeValidator
	logger         *zap.Logger
}

// NewParserWorker creates a new parser worker
func NewParserWorker(
	rmqClient rabbitmq.Client,
	parsingQueue string,
	contentStore services.ContentStore,
	taskRepo crawltask.CrawlTaskRepository,
	jobRepo crawljob.CrawlJobRepository,
	jobConfigRepo crawljobconfig.CrawlJobConfigRepository,
	outboxRepo outbox.OutboxRepository,
	txManager persistence.TxManager,
	scopeValidator services.ScopeValidator,
	logger *zap.Logger,
) *ParserWorker {
	return &ParserWorker{
		rmqClient:      rmqClient,
		parsingQueue:   parsingQueue,
		contentStore:   contentStore,
		taskRepo:       taskRepo,
		jobRepo:        jobRepo,
		jobConfigRepo:  jobConfigRepo,
		outboxRepo:     outboxRepo,
		txManager:      txManager,
		scopeValidator: scopeValidator,
		logger:         logger,
	}
}

// Start starts consuming messages from parsing_queue
func (w *ParserWorker) Start(ctx context.Context) error {
	w.logger.Info("Starting parser worker", zap.String("queue", w.parsingQueue))
	return w.rmqClient.Consume(ctx, w.parsingQueue, w.handleMessage)
}

// handleMessage processes a single parsing task message
func (w *ParserWorker) handleMessage(body []byte) error {
	// Parse message
	var task rabbitmq.ParsingTaskMessage
	if err := json.Unmarshal(body, &task); err != nil {
		w.logger.Error("Failed to unmarshal parsing task", zap.Error(err))
		return fmt.Errorf("failed to unmarshal task: %w", err)
	}

	w.logger.Info("Received parsing task",
		zap.String("task_id", task.TaskID),
		zap.String("job_id", task.JobID),
	)

	ctx := context.Background()

	// Parse task ID
	taskID, err := valueobjects.NewCrawlTaskID(task.TaskID)
	if err != nil {
		return fmt.Errorf("invalid task ID: %w", err)
	}

	// Load task from DB
	crawlTask, err := w.taskRepo.Get(ctx, taskID)
	if err != nil {
		w.logger.Error("Failed to load crawl task",
			zap.String("task_id", task.TaskID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to load crawl task: %w", err)
	}

	// Load job from DB
	crawlJob, err := w.jobRepo.Get(ctx, crawlTask.JobID)
	if err != nil {
		w.logger.Error("Failed to load crawl job",
			zap.String("job_id", task.JobID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to load crawl job: %w", err)
	}

	// Load job config from DB
	jobConfig, err := w.jobConfigRepo.Get(ctx, crawlJob.JobConfigID)
	if err != nil {
		w.logger.Error("Failed to load job config",
			zap.String("job_config_id", crawlJob.JobConfigID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to load job config: %w", err)
	}

	// Deduplication check: if we've already processed this content (same body_hash) for this job, skip
	// We exclude the current task to avoid false positive (task comparing with itself)
	if crawlTask.BodyHash != "" {
		exists, err := w.taskRepo.ExistsByJobIDAndHashExcluding(ctx, crawlTask.JobID, crawlTask.BodyHash, crawlTask.ID)
		if err != nil {
			w.logger.Error("Failed to check for duplicate body_hash",
				zap.String("task_id", task.TaskID),
				zap.String("job_id", task.JobID),
				zap.String("body_hash", crawlTask.BodyHash),
				zap.Error(err),
			)
			return fmt.Errorf("failed to check for duplicate: %w", err)
		}

		if exists {
			w.logger.Info("Skipping task - duplicate content already processed",
				zap.String("task_id", task.TaskID),
				zap.String("job_id", task.JobID),
				zap.String("body_hash", crawlTask.BodyHash),
				zap.String("url", crawlTask.URL),
			)

			// Update task status to Skipped
			crawlTask.Status = models.TaskStatusSkipped
			if err := w.taskRepo.Update(ctx, *crawlTask); err != nil {
				w.logger.Error("Failed to update task status to Skipped",
					zap.String("task_id", task.TaskID),
					zap.Error(err),
				)
				return fmt.Errorf("failed to update task status: %w", err)
			}

			// Task already processed with same content - skip without error
			return nil
		}
	}

	// Load HTML from MinIO
	htmlContent, err := w.contentStore.Get(ctx, crawlTask.MinioObjectKey)
	if err != nil {
		w.logger.Error("Failed to load HTML from MinIO",
			zap.String("task_id", task.TaskID),
			zap.String("minio_key", crawlTask.MinioObjectKey),
			zap.Error(err),
		)
		return fmt.Errorf("failed to load HTML from MinIO: %w", err)
	}

	// Parse and extract data using DSL
	result, err := w.extractData(ctx, crawlTask, jobConfig.ExtractionSpec, htmlContent)
	if err != nil {
		w.logger.Error("Failed to extract data",
			zap.String("task_id", task.TaskID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to extract data: %w", err)
	}

	// Persist results to S3 and DB (Part A - result persistence)
	if err := w.persistResults(ctx, task.TaskID, crawlTask.URL, result); err != nil {
		w.logger.Error("Failed to persist results",
			zap.String("task_id", task.TaskID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to persist results: %w", err)
	}

	// Discover and enqueue new links for crawling
	if err := w.discoverAndEnqueueLinks(ctx, crawlTask, htmlContent, jobConfig); err != nil {
		w.logger.Error("Failed to discover and enqueue links",
			zap.String("task_id", task.TaskID),
			zap.Error(err),
		)
		// Don't fail the entire task if link discovery fails
		// Just log the error and continue
	}

	return nil
}

// extractionResult holds the extracted data and computed metrics
type extractionResult struct {
	Fields  map[string]any `json:"fields"`
	Metrics map[string]any `json:"metrics"`
}

// extractData performs DSL-based extraction on HTML content
func (w *ParserWorker) extractData(
	ctx context.Context,
	task *models.CrawlTask,
	spec models.ExtractionSpec,
	htmlContent []byte,
) (*extractionResult, error) {
	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlContent)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Parse page URL for link resolution
	var baseURL *url.URL
	if task.FinalURL != nil && *task.FinalURL != "" {
		baseURL, _ = url.Parse(*task.FinalURL)
	} else {
		baseURL, _ = url.Parse(task.URL)
	}

	// Extract plain text for text-based selectors
	bodyText := doc.Find("body").Text()

	// Extract fields
	fields := make(map[string]any)
	for _, fieldSpec := range spec.Fields {
		value, err := w.extractField(fieldSpec, doc, bodyText, baseURL)
		if err != nil && fieldSpec.Required {
			w.logger.Warn("Failed to extract required field",
				zap.String("field", fieldSpec.Name),
				zap.Error(err),
			)
		}
		fields[fieldSpec.Name] = value
	}

	// Compute metrics
	metrics := make(map[string]any)
	for _, metricSpec := range spec.Metrics {
		value := w.computeMetric(metricSpec, fields, doc, bodyText)
		metrics[metricSpec.Name] = value
	}

	return &extractionResult{
		Fields:  fields,
		Metrics: metrics,
	}, nil
}

// extractField extracts a single field according to its specification
func (w *ParserWorker) extractField(
	spec models.FieldSpec,
	doc *goquery.Document,
	bodyText string,
	baseURL *url.URL,
) (any, error) {
	// Extract raw value using extractor
	rawValue, err := w.applyExtractor(spec.Extractor, doc, bodyText, baseURL)
	if err != nil {
		return nil, err
	}

	// If value is nil/empty, return nil
	if rawValue == nil || rawValue == "" {
		return nil, nil
	}

	// Apply transforms
	transformedValue := rawValue
	for _, transform := range spec.Transforms {
		transformedValue = w.applyTransform(transform, transformedValue)
	}

	// Convert to expected type
	finalValue, err := w.convertToType(transformedValue, spec.Type)
	if err != nil {
		return nil, err
	}

	return finalValue, nil
}

// applyExtractor extracts raw value from HTML using CSS selectors
func (w *ParserWorker) applyExtractor(
	spec models.ExtractorSpec,
	doc *goquery.Document,
	bodyText string,
	baseURL *url.URL,
) (any, error) {
	// Always extract from HTML using CSS selectors
	return w.extractWithCSS(spec, doc, baseURL)
}

// extractWithCSS extracts data using CSS selectors
func (w *ParserWorker) extractWithCSS(
	spec models.ExtractorSpec,
	doc *goquery.Document,
	baseURL *url.URL,
) (any, error) {
	selection := doc.Find(spec.Selector)
	if selection.Length() == 0 {
		return nil, fmt.Errorf("no elements found for selector: %s", spec.Selector)
	}

	if spec.Multiple {
		var results []string
		selection.Each(func(i int, s *goquery.Selection) {
			value := w.extractElementValue(s, spec.Attribute, baseURL)
			if value != "" {
				results = append(results, value)
			}
		})

		// If index specified, return that element
		if spec.Index != nil {
			idx := *spec.Index
			if idx < 0 {
				idx = len(results) + idx // -1 => last
			}
			if idx >= 0 && idx < len(results) {
				return results[idx], nil
			}
		}

		return results, nil
	}

	// Single value
	return w.extractElementValue(selection.First(), spec.Attribute, baseURL), nil
}

// extractElementValue extracts value from a goquery selection
func (w *ParserWorker) extractElementValue(
	selection *goquery.Selection,
	attribute string,
	baseURL *url.URL,
) string {
	attr := strings.ToLower(strings.TrimSpace(attribute))

	// ✅ Псевдо-атрибуты: "text" / "html"
	switch attr {
	case "", "text", "innertext":
		return strings.TrimSpace(selection.Text())
	case "html", "innerhtml":
		h, err := selection.Html()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(h)
	}

	// Обычные HTML-атрибуты
	value, exists := selection.Attr(attribute)
	if !exists {
		return ""
	}

	value = strings.TrimSpace(value)

	// Resolve relative URLs for href/src attributes
	if (attr == "href" || attr == "src") && baseURL != nil {
		if resolved, err := baseURL.Parse(value); err == nil {
			return resolved.String()
		}
	}

	return value
}

// applyTransform applies a single transform operation to a value
func (w *ParserWorker) applyTransform(spec models.TransformSpec, value any) any {
	if value == nil {
		return nil
	}

	switch spec.Op {
	case models.OpTrim:
		if str, ok := value.(string); ok {
			return strings.TrimSpace(str)
		}
	case models.OpLower:
		if str, ok := value.(string); ok {
			return strings.ToLower(str)
		}
	case models.OpUpper:
		if str, ok := value.(string); ok {
			return strings.ToUpper(str)
		}
	case models.OpCollapseWS:
		if str, ok := value.(string); ok {
			return regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(str), " ")
		}
	case models.OpHTMLToText:
		if str, ok := value.(string); ok {
			// Simple HTML tag removal
			return regexp.MustCompile(`<[^>]*>`).ReplaceAllString(str, "")
		}
	case models.OpNormalizeURL:
		if str, ok := value.(string); ok {
			if parsed, err := url.Parse(str); err == nil {
				return parsed.String()
			}
		}
	case models.OpUnique:
		if arr, ok := value.([]string); ok {
			seen := make(map[string]bool)
			var unique []string
			for _, item := range arr {
				if !seen[item] {
					seen[item] = true
					unique = append(unique, item)
				}
			}
			return unique
		}
	case models.OpLimit:
		if arr, ok := value.([]string); ok {
			// arg может быть int / float64 / json.Number / string
			limit := -1
			switch v := spec.Arg.(type) {
			case int:
				limit = v
			case int64:
				limit = int(v)
			case float64:
				limit = int(v)
			case json.Number:
				if i, err := v.Int64(); err == nil {
					limit = int(i)
				}
			case string:
				if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
					limit = i
				}
			}

			if limit >= 0 && limit < len(arr) {
				return arr[:limit]
			}
		}
	case models.OpParseInt:
		if str, ok := value.(string); ok {
			if i, err := strconv.ParseInt(strings.TrimSpace(str), 10, 64); err == nil {
				return i
			}
		}
	case models.OpParseFloat:
		if str, ok := value.(string); ok {
			if f, err := strconv.ParseFloat(strings.TrimSpace(str), 64); err == nil {
				return f
			}
		}
	case models.OpParsePrice:
		if str, ok := value.(string); ok {
			// Extract numeric value from price string (e.g., "$19.99" -> 19.99)
			re := regexp.MustCompile(`\d+\.?\d*`)
			if match := re.FindString(str); match != "" {
				if f, err := strconv.ParseFloat(match, 64); err == nil {
					return f
				}
			}
		}
	case models.OpSHA256:
		if str, ok := value.(string); ok {
			hash := sha256.Sum256([]byte(str))
			return fmt.Sprintf("%x", hash)
		}
	}

	return value
}

// convertToType converts a value to the expected type
func (w *ParserWorker) convertToType(value any, valueType models.ValueType) (any, error) {
	switch valueType {
	case models.ValueString:
		return fmt.Sprintf("%v", value), nil
	case models.ValueInt:
		switch v := value.(type) {
		case int:
			return int64(v), nil
		case int64:
			return v, nil
		case float64:
			return int64(v), nil
		case json.Number:
			return v.Int64()
		case string:
			return strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		}
	case models.ValueFloat:
		switch v := value.(type) {
		case float64:
			return v, nil
		case float32:
			return float64(v), nil
		case int:
			return float64(v), nil
		case int64:
			return float64(v), nil
		case string:
			return strconv.ParseFloat(strings.TrimSpace(v), 64)
		}
	case models.ValueBool:
		switch v := value.(type) {
		case bool:
			return v, nil
		case string:
			return strconv.ParseBool(v)
		}
	case models.ValueURL:
		if str, ok := value.(string); ok {
			_, err := url.Parse(str)
			if err != nil {
				return nil, err
			}
			return str, nil
		}
	case models.ValueJSON:
		// Already in native Go format
		return value, nil
	}

	return value, nil
}

// computeMetric computes a single metric value
func (w *ParserWorker) computeMetric(
	spec models.MetricSpec,
	fields map[string]any,
	doc *goquery.Document,
	bodyText string,
) any {
	switch spec.Op {
	case models.MetricLen:
		if value, ok := fields[spec.Input]; ok {
			if str, ok := value.(string); ok {
				return len(str)
			}
		}
		return 0

	case models.MetricCount:
		value, ok := fields[spec.Input]
		if !ok || value == nil {
			return 0
		}

		countNonEmpty := func(ss []string) int {
			n := 0
			for _, s := range ss {
				if strings.TrimSpace(s) != "" {
					n++
				}
			}
			return n
		}

		switch v := value.(type) {
		case []string:
			return countNonEmpty(v)
		case []any:
			tmp := make([]string, 0, len(v))
			for _, it := range v {
				s := strings.TrimSpace(fmt.Sprintf("%v", it))
				if s != "" {
					tmp = append(tmp, s)
				}
			}
			return len(tmp)
		default:
			return 0
		}

	case models.MetricWordCount:
		var text string
		if spec.Input == "body_text" {
			text = bodyText
		} else if value, ok := fields[spec.Input]; ok {
			text = fmt.Sprintf("%v", value)
		}
		words := strings.Fields(text)
		return len(words)

	case models.MetricFieldPresent:
		if value, ok := fields[spec.Input]; ok {
			return value != nil && value != ""
		}
		return false

	case models.MetricCountExternalLinks:
		count := 0
		doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
			if href, exists := s.Attr("href"); exists {
				if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
					count++
				}
			}
		})
		return count
	}

	return nil
}

// persistResults uploads extraction results to S3 and updates DB (Part A - result persistence)
func (w *ParserWorker) persistResults(ctx context.Context, taskID, url string, result *extractionResult) error {
	// Prepare output structure (same as before)
	output := map[string]any{
		"task_id": taskID,
		"url":     url,
		"fields":  result.Fields,
		"metrics": result.Metrics,
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	// Determine S3 object key: results/tasks/{task_id}.json
	objectKey := fmt.Sprintf("results/tasks/%s.json", taskID)

	// Upload to S3
	if err := w.contentStore.Store(ctx, objectKey, jsonData, "application/json"); err != nil {
		return fmt.Errorf("failed to upload result to S3: %w", err)
	}

	w.logger.Info("Result uploaded to S3",
		zap.String("task_id", taskID),
		zap.String("object_key", objectKey),
		zap.Int("size_bytes", len(jsonData)),
	)

	// Update DB with result reference
	parsedTaskID, err := valueobjects.NewCrawlTaskID(taskID)
	if err != nil {
		return fmt.Errorf("invalid task ID: %w", err)
	}

	if err := w.taskRepo.SetTaskResult(ctx, parsedTaskID, objectKey, "application/json", int64(len(jsonData))); err != nil {
		return fmt.Errorf("failed to update task result in DB: %w", err)
	}

	w.logger.Info("Task result reference saved to DB",
		zap.String("task_id", taskID),
		zap.String("result_object_key", objectKey),
	)

	return nil
}

// discoverAndEnqueueLinks extracts links from HTML, filters them, and enqueues new crawl tasks via Outbox
func (w *ParserWorker) discoverAndEnqueueLinks(
	ctx context.Context,
	crawlTask *models.CrawlTask,
	htmlContent []byte,
	jobConfig *models.CrawlJobConfig,
) error {
	// Check depth limit - stop if we've reached MaxDepth
	if crawlTask.Depth >= jobConfig.Scopes.MaxDepth {
		w.logger.Debug("Max depth reached, skipping link discovery",
			zap.String("task_id", crawlTask.ID.String()),
			zap.Uint64("current_depth", crawlTask.Depth),
			zap.Uint64("max_depth", jobConfig.Scopes.MaxDepth),
		)
		return nil
	}

	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlContent)))
	if err != nil {
		return fmt.Errorf("failed to parse HTML for link discovery: %w", err)
	}

	// Determine base URL for resolving relative links
	var baseURL *url.URL
	if crawlTask.FinalURL != nil && *crawlTask.FinalURL != "" {
		baseURL, err = url.Parse(*crawlTask.FinalURL)
	} else {
		baseURL, err = url.Parse(crawlTask.URL)
	}
	if err != nil {
		return fmt.Errorf("failed to parse base URL: %w", err)
	}

	// Extract all links and dedupe
	linkSet := make(map[string]bool)
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		// Parse and resolve relative URL
		linkURL, err := url.Parse(strings.TrimSpace(href))
		if err != nil {
			return
		}

		// Resolve relative URLs
		absoluteURL := baseURL.ResolveReference(linkURL)

		// Filter out unwanted schemes
		scheme := strings.ToLower(absoluteURL.Scheme)
		if scheme != "http" && scheme != "https" {
			return
		}

		// Remove fragment
		absoluteURL.Fragment = ""

		normalizedURL := absoluteURL.String()
		linkSet[normalizedURL] = true
	})

	if len(linkSet) == 0 {
		w.logger.Debug("No valid links found on page",
			zap.String("task_id", crawlTask.ID.String()),
		)
		return nil
	}

	// Filter links by scope rules and prepare tasks/events
	nextDepth := crawlTask.Depth + 1
	tasks := make([]models.CrawlTask, 0)
	outboxEvents := make([]models.OutboxEvent, 0)
	now := time.Now().UTC()

	for link := range linkSet {
		// Validate against scope rules
		if err := w.scopeValidator.Validate(link, nextDepth, jobConfig.Scopes); err != nil {
			w.logger.Debug("Link filtered by scope rules",
				zap.String("url", link),
				zap.Error(err),
			)
			continue
		}

		// Create new CrawlTask
		taskID := valueobjects.GenerateCrawlTaskID()
		task := models.CrawlTask{
			ID:         taskID,
			JobID:      crawlTask.JobID,
			URL:        link,
			Status:     models.TaskStatusInProgress,
			EnqueuedAt: now,
			Depth:      nextDepth,
		}
		tasks = append(tasks, task)

		// Create TaskEnqueuedEvent
		event := events.NewTaskEnqueuedEvent(
			taskID.String(),
			crawlTask.JobID.String(),
			link,
			now,
		)

		// Marshal event to JSON
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event for task %s: %w", taskID.String(), err)
		}

		// Create OutboxEvent
		outboxEvent := models.OutboxEvent{
			ID:          valueobjects.GenerateOutboxEventID(),
			EventType:   string(event.Type),
			AggregateID: taskID.String(),
			Payload:     payload,
			OccurredAt:  event.OccurredAt,
			ProcessedAt: nil,
			CreatedAt:   now,
		}
		outboxEvents = append(outboxEvents, outboxEvent)
	}

	if len(tasks) == 0 {
		w.logger.Debug("No links passed scope validation",
			zap.String("task_id", crawlTask.ID.String()),
			zap.Int("total_links", len(linkSet)),
		)
		return nil
	}

	// Persist tasks and outbox events atomically
	err = w.txManager.ReadCommitted(ctx, func(ctx context.Context) error {
		// Bulk create tasks
		if err := w.taskRepo.BulkCreate(ctx, tasks); err != nil {
			return fmt.Errorf("failed to bulk create tasks: %w", err)
		}

		// Bulk create outbox events
		if err := w.outboxRepo.BulkCreate(ctx, outboxEvents); err != nil {
			return fmt.Errorf("failed to bulk create outbox events: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to persist discovered links: %w", err)
	}

	w.logger.Info("Successfully discovered and enqueued new links",
		zap.String("task_id", crawlTask.ID.String()),
		zap.Int("links_discovered", len(linkSet)),
		zap.Int("links_enqueued", len(tasks)),
		zap.Uint64("next_depth", nextDepth),
	)

	return nil
}
