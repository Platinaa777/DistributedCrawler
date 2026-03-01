package worker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"distributed-crawler/internal/domain/crawl/events"
	"distributed-crawler/internal/domain/crawl/models"
	crawljob "distributed-crawler/internal/domain/crawl/repos/crawl_job"
	crawljobconfig "distributed-crawler/internal/domain/crawl/repos/crawl_job_config"
	crawltask "distributed-crawler/internal/domain/crawl/repos/crawl_task"
	"distributed-crawler/internal/domain/crawl/repos/outbox"
	"distributed-crawler/internal/domain/crawl/services"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/messaging"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/telemetry"

	"github.com/PuerkitoBio/goquery"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	htmlcharset "golang.org/x/net/html/charset"
)

// ParserWorker consumes parsing tasks, loads HTML from MinIO, parses using DSL, and prints results
type ParserWorker struct {
	msgClient        messaging.Client
	parsingQueue     string
	contentStore     services.ContentStore
	taskRepo         crawltask.CrawlTaskRepository
	jobRepo          crawljob.CrawlJobRepository
	jobConfigRepo    crawljobconfig.CrawlJobConfigRepository
	outboxRepo       outbox.OutboxRepository
	txManager        persistence.TxManager
	scopeValidator   services.ScopeValidator
	robotsTxtService services.RobotsTxtService
	logger           *zap.Logger
	activeTasks      atomic.Int64

	tracer   trace.Tracer
	metrics  *telemetry.Metrics
	workerID string
}

// NewParserWorker creates a new parser worker
func NewParserWorker(
	msgClient messaging.Client,
	parsingQueue string,
	contentStore services.ContentStore,
	taskRepo crawltask.CrawlTaskRepository,
	jobRepo crawljob.CrawlJobRepository,
	jobConfigRepo crawljobconfig.CrawlJobConfigRepository,
	outboxRepo outbox.OutboxRepository,
	txManager persistence.TxManager,
	scopeValidator services.ScopeValidator,
	robotsTxtService services.RobotsTxtService,
	logger *zap.Logger,
	tracer trace.Tracer,
	metrics *telemetry.Metrics,
	workerID string,
) *ParserWorker {
	return &ParserWorker{
		msgClient:        msgClient,
		parsingQueue:     parsingQueue,
		contentStore:     contentStore,
		taskRepo:         taskRepo,
		jobRepo:          jobRepo,
		jobConfigRepo:    jobConfigRepo,
		outboxRepo:       outboxRepo,
		txManager:        txManager,
		scopeValidator:   scopeValidator,
		robotsTxtService: robotsTxtService,
		logger:           logger,
		tracer:           tracer,
		metrics:          metrics,
		workerID:         workerID,
	}
}

// Start starts consuming messages from parsing_queue
func (w *ParserWorker) Start(ctx context.Context) error {
	w.logger.Info("Starting parser worker", zap.String("queue", w.parsingQueue))
	return w.msgClient.Consume(ctx, w.parsingQueue, w.handleMessage)
}

// ActiveTasks returns the number of tasks currently being processed.
func (w *ParserWorker) ActiveTasks() int32 {
	return int32(w.activeTasks.Load())
}

// handleMessage processes a single parsing task message
func (w *ParserWorker) handleMessage(body []byte) error {
	w.activeTasks.Add(1)
	defer w.activeTasks.Add(-1)

	startTime := time.Now()

	// Parse message
	var task messaging.ParsingTaskMessage
	if err := json.Unmarshal(body, &task); err != nil {
		w.logger.Error("Failed to unmarshal parsing task", zap.Error(err))
		return fmt.Errorf("failed to unmarshal task: %w", err)
	}

	w.logger.Info("Received parsing task",
		zap.String("task_id", task.TaskID),
		zap.String("job_id", task.JobID),
	)

	// Extract trace context from message and create span
	ctx := telemetry.ExtractTraceContext(context.Background(), task.TraceContext)

	var span trace.Span
	if w.tracer != nil {
		ctx, span = w.tracer.Start(ctx, "parser_worker_process",
			trace.WithSpanKind(trace.SpanKindConsumer),
			trace.WithAttributes(
				attribute.String("job.id", task.JobID),
				attribute.String("task.id", task.TaskID),
				attribute.String("worker.id", w.workerID),
				attribute.String("worker.type", "parser"),
			),
		)
		defer span.End()
	}

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

	// Idempotency guard: skip if already parsed (e.g. RMQ retry after successful processing)
	if crawlTask.Status == models.TaskStatusParsed {
		w.logger.Info("Task already parsed, skipping",
			zap.String("task_id", task.TaskID),
		)
		return nil
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

		// Mark task as failed with error message
		crawlTask.Status = models.TaskStatusFailed
		errMsg := fmt.Sprintf("extraction failed: %v", err)
		crawlTask.ErrorMessage = &errMsg
		if updateErr := w.taskRepo.Update(ctx, *crawlTask); updateErr != nil {
			w.logger.Error("Failed to update task status to failed",
				zap.String("task_id", task.TaskID),
				zap.Error(updateErr),
			)
			return fmt.Errorf("failed to update task status: %w", updateErr)
		}

		w.logger.Info("Task marked as failed due to extraction error",
			zap.String("task_id", task.TaskID),
			zap.String("url", crawlTask.URL),
		)

		// Don't return error - task is processed (marked as failed)
		return nil
	}

	// Determine effective crawl mode (default to pagination_and_links when unset)
	crawlMode := jobConfig.CrawlMode
	if crawlMode == "" {
		crawlMode = models.CrawlModePaginationAndLinks
	}

	// Upload extraction result to S3 first (outside transaction — idempotent).
	objectKey, sizeBytes, err := w.uploadResultToS3(ctx, task.TaskID, crawlTask.URL, result)
	if err != nil {
		w.logger.Error("Failed to upload result to S3",
			zap.String("task_id", task.TaskID),
			zap.Error(err),
		)

		crawlTask.Status = models.TaskStatusFailed
		errMsg := fmt.Sprintf("failed to persist results: %v", err)
		crawlTask.ErrorMessage = &errMsg
		if updateErr := w.taskRepo.Update(ctx, *crawlTask); updateErr != nil {
			w.logger.Error("Failed to update task status to failed",
				zap.String("task_id", task.TaskID),
				zap.Error(updateErr),
			)
			return fmt.Errorf("failed to update task status: %w", updateErr)
		}

		w.logger.Info("Task marked as failed due to persist error",
			zap.String("task_id", task.TaskID),
			zap.String("url", crawlTask.URL),
		)

		return nil
	}

	// Prepare new tasks/events outside any transaction (robots.txt checks involve network I/O).
	var paginationTasks []models.CrawlTask
	var paginationEvents []models.OutboxEvent
	if crawlMode != models.CrawlModeLinksOnly {
		paginationTasks, paginationEvents, err = w.preparePaginationLinks(ctx, crawlTask, htmlContent, jobConfig)
		if err != nil {
			w.logger.Error("Failed to prepare pagination links",
				zap.String("task_id", task.TaskID),
				zap.Error(err),
			)
		}
	}

	var discoveryTasks []models.CrawlTask
	var discoveryEvents []models.OutboxEvent
	if crawlMode != models.CrawlModePaginationOnly {
		discoveryTasks, discoveryEvents, err = w.prepareDiscoveredLinks(ctx, crawlTask, htmlContent, jobConfig)
		if err != nil {
			w.logger.Error("Failed to prepare discovered links",
				zap.String("task_id", task.TaskID),
				zap.Error(err),
			)
		}
	}

	// Atomically: persist result DB reference + all new tasks + outbox events.
	// A single transaction closes the race window where the export worker could
	// see result_object_key set on the last task but find no pending outbox events
	// for newly discovered URLs, causing it to export an incomplete job.
	allTasks := append(paginationTasks, discoveryTasks...)
	allEvents := append(paginationEvents, discoveryEvents...)

	if err := w.txManager.ReadCommitted(ctx, func(ctx context.Context) error {
		crawlTask.MarkAsParsed(objectKey, "application/json", sizeBytes, time.Now())

		if err := w.taskRepo.Update(ctx, *crawlTask); err != nil {
			return fmt.Errorf("failed to update task status to parsed: %w", err)
		}

		eventsToCreate := allEvents
		if len(allTasks) > 0 {
			insertedIDs, err := w.taskRepo.BulkCreate(ctx, allTasks)
			if err != nil {
				return fmt.Errorf("failed to bulk create new tasks: %w", err)
			}
			// BulkCreate uses ON CONFLICT DO NOTHING: some tasks may have been skipped because
			// the (job_id, url) pair already exists. Only create outbox events for tasks that
			// were actually inserted — otherwise the fetch worker would receive a message for a
			// task_id that is not in the database.
			if len(insertedIDs) < len(allTasks) {
				insertedSet := make(map[string]bool, len(insertedIDs))
				for _, id := range insertedIDs {
					insertedSet[id.String()] = true
				}
				filtered := make([]models.OutboxEvent, 0, len(insertedIDs))
				for _, e := range eventsToCreate {
					if insertedSet[e.AggregateID] {
						filtered = append(filtered, e)
					}
				}
				eventsToCreate = filtered
			}
		}

		if len(eventsToCreate) > 0 {
			if err := w.outboxRepo.BulkCreate(ctx, eventsToCreate); err != nil {
				return fmt.Errorf("failed to bulk create outbox events: %w", err)
			}
		}
		return nil
	}); err != nil {
		w.logger.Error("Failed to persist results and new tasks",
			zap.String("task_id", task.TaskID),
			zap.Error(err),
		)

		crawlTask.Status = models.TaskStatusFailed
		errMsg := fmt.Sprintf("failed to persist results: %v", err)
		crawlTask.ErrorMessage = &errMsg
		if updateErr := w.taskRepo.Update(ctx, *crawlTask); updateErr != nil {
			w.logger.Error("Failed to update task status to failed",
				zap.String("task_id", task.TaskID),
				zap.Error(updateErr),
			)
			return fmt.Errorf("failed to update task status: %w", updateErr)
		}

		w.logger.Info("Task marked as failed due to persist error",
			zap.String("task_id", task.TaskID),
			zap.String("url", crawlTask.URL),
		)

		return nil
	}

	// Record successful parsing duration metric
	duration := time.Since(startTime).Seconds()
	w.recordParserDuration(ctx, "text/html", duration)

	// Record task completion metric
	w.recordTaskCompleted(ctx)

	// Set span attributes for successful completion
	if span != nil {
		span.SetAttributes(
			attribute.Int("page.size_bytes", len(htmlContent)),
			attribute.Int("parser.fields_extracted", len(result.Fields)),
			attribute.Int("parser.items_extracted", len(result.Items)),
		)
		span.SetStatus(codes.Ok, "")
	}

	w.logger.Info("Successfully processed parsing task",
		zap.String("task_id", task.TaskID),
		zap.String("url", crawlTask.URL),
		zap.Duration("duration", time.Since(startTime)),
	)

	return nil
}

// recordParserDuration records the parser duration metric
func (w *ParserWorker) recordParserDuration(ctx context.Context, contentType string, duration float64) {
	if w.metrics == nil || w.metrics.ParserDuration == nil {
		return
	}
	w.metrics.ParserDuration.Record(ctx, duration,
		metric.WithAttributes(
			attribute.String("content_type", contentType),
		),
	)
}

// recordTaskCompleted records the task completed metric
func (w *ParserWorker) recordTaskCompleted(ctx context.Context) {
	if w.metrics == nil || w.metrics.TasksCompletedTotal == nil {
		return
	}
	w.metrics.TasksCompletedTotal.Add(ctx, 1)
}

// recordTaskFailed records the task failed metric
func (w *ParserWorker) recordTaskFailed(ctx context.Context, reason string) {
	if w.metrics == nil || w.metrics.TasksFailedTotal == nil {
		return
	}
	w.metrics.TasksFailedTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("failure_reason", reason),
		),
	)
}

// extractionResult holds the extracted data
type extractionResult struct {
	Fields map[string]any   `json:"fields,omitempty"`
	Items  []map[string]any `json:"items,omitempty"`
}

// extractData performs DSL-based extraction on HTML content
func (w *ParserWorker) extractData(
	ctx context.Context,
	task *models.CrawlTask,
	spec models.ExtractionSpec,
	htmlContent []byte,
) (*extractionResult, error) {
	// Parse HTML with goquery
	doc, err := w.parseHTMLDocument(htmlContent)
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

	// Extract page-level fields
	var fields map[string]any
	if len(spec.Fields) > 0 {
		fields = make(map[string]any)
		for _, fieldSpec := range spec.Fields {
			value, err := w.extractField(fieldSpec, doc, baseURL)
			if err != nil && fieldSpec.Required {
				w.logger.Warn("Failed to extract required field",
					zap.String("field", fieldSpec.Name),
					zap.Error(err),
				)
			}
			fields[fieldSpec.Name] = value
		}
	}

	// Extract structured items if ItemsSpec is defined
	var items []map[string]any
	if spec.Items != nil {
		items = w.extractItems(spec.Items, doc, baseURL)
	}

	return &extractionResult{
		Fields: fields,
		Items:  items,
	}, nil
}

// extractField extracts a single field according to its specification
func (w *ParserWorker) extractField(
	spec models.FieldSpec,
	doc *goquery.Document,
	baseURL *url.URL,
) (any, error) {
	// Extract raw value using extractor
	rawValue, err := w.applyExtractor(spec.Extractor, doc, baseURL)
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

// extractItems extracts a list of structured objects from the page using ItemsSpec.
// Each element matching ContainerSelector is scoped as an isolated DOM subtree,
// and Fields are extracted relative to that container.
func (w *ParserWorker) extractItems(
	spec *models.ItemsSpec,
	doc *goquery.Document,
	baseURL *url.URL,
) []map[string]any {
	items := make([]map[string]any, 0)

	if spec.ContainerSelector == "" {
		w.logger.Warn("ItemsSpec has empty container_selector, skipping items extraction")
		return items
	}

	containers := doc.Find(spec.ContainerSelector)
	if containers.Length() == 0 {
		w.logger.Debug("No containers found for items extraction",
			zap.String("selector", spec.ContainerSelector),
		)
		return items
	}

	containers.Each(func(i int, container *goquery.Selection) {
		item := make(map[string]any)

		for _, fieldSpec := range spec.Fields {
			value, err := w.extractFieldFromSelection(fieldSpec, container, baseURL)
			if err != nil {
				if fieldSpec.Required {
					w.logger.Warn("Failed to extract required item field",
						zap.String("field", fieldSpec.Name),
						zap.Int("item_index", i),
						zap.Error(err),
					)
				}
				// Continue processing other fields and items
				continue
			}
			item[fieldSpec.Name] = value
		}

		items = append(items, item)
	})

	return items
}

// extractFieldFromSelection extracts a single field from a goquery.Selection (scoped DOM subtree).
// This is used for per-item field extraction where selectors are relative to a container element.
func (w *ParserWorker) extractFieldFromSelection(
	spec models.FieldSpec,
	sel *goquery.Selection,
	baseURL *url.URL,
) (any, error) {
	rawValue, err := w.extractWithCSSFromSelection(spec.Extractor, sel, baseURL)
	if err != nil {
		return nil, err
	}

	if rawValue == nil || rawValue == "" {
		return nil, nil
	}

	transformedValue := rawValue
	for _, transform := range spec.Transforms {
		transformedValue = w.applyTransform(transform, transformedValue)
	}

	return w.convertToType(transformedValue, spec.Type)
}

// extractWithCSSFromSelection extracts data using CSS selectors relative to a goquery.Selection.
// This mirrors extractWithCSS but scoped to a subtree instead of the full document.
func (w *ParserWorker) extractWithCSSFromSelection(
	spec models.ExtractorSpec,
	sel *goquery.Selection,
	baseURL *url.URL,
) (any, error) {
	selection := sel.Find(spec.Selector)
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

		if spec.Index != nil {
			idx := *spec.Index
			if idx < 0 {
				idx = len(results) + idx
			}
			if idx >= 0 && idx < len(results) {
				return results[idx], nil
			}
		}

		return results, nil
	}

	return w.extractElementValue(selection.First(), spec.Attribute, baseURL), nil
}

// applyExtractor extracts raw value from HTML using CSS selectors
func (w *ParserWorker) applyExtractor(
	spec models.ExtractorSpec,
	doc *goquery.Document,
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
	value, exists := selection.Attr(attr)
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

func (w *ParserWorker) parseHTMLDocument(htmlContent []byte) (*goquery.Document, error) {
	reader, err := htmlcharset.NewReader(bytes.NewReader(htmlContent), "")
	if err != nil {
		if w.logger != nil {
			w.logger.Debug("Failed to detect HTML charset, falling back to raw bytes",
				zap.Error(err),
			)
		}
		reader = bytes.NewReader(htmlContent)
	}

	return goquery.NewDocumentFromReader(reader)
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
		switch v := value.(type) {
		case []string:
			// keep list results as-is so metrics and callers can treat as array
			return v, nil
		case []any:
			out := make([]string, 0, len(v))
			for _, it := range v {
				out = append(out, fmt.Sprintf("%v", it))
			}
			return out, nil
		default:
			return fmt.Sprintf("%v", value), nil
		}
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

// uploadResultToS3 marshals the extraction result as JSON and stores it in S3.
// It returns the object key and byte size so the caller can later persist the
// DB reference atomically alongside newly-discovered task records.
// The current trace context is embedded in the JSON so the export worker can
// create span links back to this parser span for e2e trace visibility.
func (w *ParserWorker) uploadResultToS3(ctx context.Context, taskID, url string, result *extractionResult) (string, int64, error) {
	// Prepare output structure
	output := map[string]any{
		"task_id":       taskID,
		"url":           url,
		"trace_context": telemetry.InjectTraceContext(ctx),
	}
	if len(result.Fields) > 0 {
		output["fields"] = result.Fields
	}
	if len(result.Items) > 0 {
		output["items"] = result.Items
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal output: %w", err)
	}

	// Determine S3 object key: results/tasks/{task_id}.json
	objectKey := fmt.Sprintf("results/tasks/%s.json", taskID)

	// Upload to S3
	if err := w.contentStore.Store(ctx, objectKey, jsonData, "application/json"); err != nil {
		return "", 0, fmt.Errorf("failed to upload result to S3: %w", err)
	}

	w.logger.Info("Result uploaded to S3",
		zap.String("task_id", taskID),
		zap.String("object_key", objectKey),
		zap.Int("size_bytes", len(jsonData)),
	)

	return objectKey, int64(len(jsonData)), nil
}

// preparePaginationLinks extracts URLs from pagination elements using user-defined CSS selectors
// and returns the CrawlTask and OutboxEvent records to be created. The caller is responsible
// for persisting them — typically in the same transaction as SetTaskResult.
func (w *ParserWorker) preparePaginationLinks(
	ctx context.Context,
	crawlTask *models.CrawlTask,
	htmlContent []byte,
	jobConfig *models.CrawlJobConfig,
) ([]models.CrawlTask, []models.OutboxEvent, error) {
	allowedURLPatterns, err := models.CompileAllowedURLPatterns(jobConfig.Scopes.AllowedURLPatterns)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid scopes.allowed_url_patterns: %w", err)
	}

	// Skip if no pagination selectors defined
	if len(jobConfig.ExtractionSpec.Pagination) == 0 {
		return nil, nil, nil
	}

	// Check depth limit - stop if we've reached MaxDepth
	if crawlTask.Depth >= jobConfig.Scopes.MaxDepth {
		w.logger.Debug("Max depth reached, skipping pagination extraction",
			zap.String("task_id", crawlTask.ID.String()),
			zap.Uint64("current_depth", crawlTask.Depth),
			zap.Uint64("max_depth", jobConfig.Scopes.MaxDepth),
		)
		return nil, nil, nil
	}

	// Parse HTML with goquery
	doc, err := w.parseHTMLDocument(htmlContent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse HTML for pagination extraction: %w", err)
	}

	// Determine base URL for resolving relative links
	var baseURL *url.URL
	if crawlTask.FinalURL != nil && *crawlTask.FinalURL != "" {
		baseURL, err = url.Parse(*crawlTask.FinalURL)
	} else {
		baseURL, err = url.Parse(crawlTask.URL)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	// Extract URLs from all pagination selectors and deduplicate
	paginationURLs := make(map[string]bool)

	for _, paginationSpec := range jobConfig.ExtractionSpec.Pagination {
		if paginationSpec.Selector == "" {
			continue
		}

		// Default attribute is "href" for pagination elements
		attribute := paginationSpec.Attribute
		if attribute == "" {
			attribute = "href"
		}

		selection := doc.Find(paginationSpec.Selector)
		if selection.Length() == 0 {
			w.logger.Debug("No elements found for pagination selector",
				zap.String("task_id", crawlTask.ID.String()),
				zap.String("selector", paginationSpec.Selector),
				zap.String("name", paginationSpec.Name),
			)
			continue
		}

		// Extract URLs from matching elements
		extractURL := func(s *goquery.Selection) {
			rawValue := w.extractElementValue(s, attribute, baseURL)
			if rawValue == "" {
				return
			}

			// Parse and resolve relative URL
			linkURL, err := url.Parse(strings.TrimSpace(rawValue))
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
			paginationURLs[normalizedURL] = true
		}

		if paginationSpec.Multiple {
			selection.Each(func(i int, s *goquery.Selection) {
				extractURL(s)
			})
		} else {
			extractURL(selection.First())
		}
	}

	if len(paginationURLs) == 0 {
		w.logger.Debug("No pagination URLs extracted",
			zap.String("task_id", crawlTask.ID.String()),
			zap.Int("pagination_selectors", len(jobConfig.ExtractionSpec.Pagination)),
		)
		return nil, nil, nil
	}

	// Filter URLs by scope rules and prepare tasks/events
	nextDepth := crawlTask.Depth + 1
	tasks := make([]models.CrawlTask, 0)
	outboxEvents := make([]models.OutboxEvent, 0)
	now := time.Now().UTC()

	// User-agent for robots.txt checking (matches the fetcher's user-agent)
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	// Capture the current parser span's trace context once for all child events.
	currentTraceCtx := telemetry.InjectTraceContext(ctx)

	for link := range paginationURLs {
		// Validate against scope rules
		if err := w.scopeValidator.Validate(link, nextDepth, jobConfig.Scopes); err != nil {
			w.logger.Debug("Pagination link filtered by scope rules",
				zap.String("url", link),
				zap.Error(err),
			)
			continue
		}
		if len(allowedURLPatterns) > 0 {
			matched := false
			for _, pattern := range allowedURLPatterns {
				if pattern.MatchString(link) {
					matched = true
					break
				}
			}
			if !matched {
				w.logger.Debug("Pagination link filtered by allowed_url_patterns",
					zap.String("url", link),
					zap.Strings("allowed_url_patterns", jobConfig.Scopes.AllowedURLPatterns),
				)
				continue
			}
		}

		// Check robots.txt rules
		allowed, err := w.robotsTxtService.IsAllowed(ctx, link, userAgent)
		if err != nil {
			w.logger.Debug("Failed to check robots.txt for pagination link, allowing",
				zap.String("url", link),
				zap.Error(err),
			)
			// On error, proceed with enqueueing (permissive default)
		} else if !allowed {
			w.logger.Debug("Pagination link disallowed by robots.txt",
				zap.String("url", link),
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

		// Create TaskEnqueuedEvent with the current parser span's trace context
		// so downstream fetch/parse spans are children of this parser span.
		event := events.NewTaskEnqueuedEvent(
			taskID.String(),
			crawlTask.JobID.String(),
			link,
			now,
		)
		event.TraceContext = currentTraceCtx

		// Marshal event to JSON
		payload, err := json.Marshal(event)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal event for pagination task %s: %w", taskID.String(), err)
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
		w.logger.Debug("No pagination links passed scope validation",
			zap.String("task_id", crawlTask.ID.String()),
			zap.Int("total_pagination_urls", len(paginationURLs)),
		)
		return nil, nil, nil
	}

	w.logger.Debug("Prepared pagination links",
		zap.String("task_id", crawlTask.ID.String()),
		zap.Int("pagination_urls_found", len(paginationURLs)),
		zap.Int("pagination_links_prepared", len(tasks)),
		zap.Uint64("next_depth", nextDepth),
	)

	return tasks, outboxEvents, nil
}

// prepareDiscoveredLinks extracts links from HTML, filters them, and returns the CrawlTask and
// OutboxEvent records to be created. The caller is responsible for persisting them — typically
// in the same transaction as SetTaskResult.
func (w *ParserWorker) prepareDiscoveredLinks(
	ctx context.Context,
	crawlTask *models.CrawlTask,
	htmlContent []byte,
	jobConfig *models.CrawlJobConfig,
) ([]models.CrawlTask, []models.OutboxEvent, error) {
	allowedURLPatterns, err := models.CompileAllowedURLPatterns(jobConfig.Scopes.AllowedURLPatterns)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid scopes.allowed_url_patterns: %w", err)
	}

	// Check depth limit - stop if we've reached MaxDepth
	if crawlTask.Depth >= jobConfig.Scopes.MaxDepth {
		w.logger.Debug("Max depth reached, skipping link discovery",
			zap.String("task_id", crawlTask.ID.String()),
			zap.Uint64("current_depth", crawlTask.Depth),
			zap.Uint64("max_depth", jobConfig.Scopes.MaxDepth),
		)
		return nil, nil, nil
	}

	// Parse HTML with goquery
	doc, err := w.parseHTMLDocument(htmlContent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse HTML for link discovery: %w", err)
	}

	// Determine base URL for resolving relative links
	var baseURL *url.URL
	if crawlTask.FinalURL != nil && *crawlTask.FinalURL != "" {
		baseURL, err = url.Parse(*crawlTask.FinalURL)
	} else {
		baseURL, err = url.Parse(crawlTask.URL)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse base URL: %w", err)
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
		return nil, nil, nil
	}

	// Filter links by scope rules and prepare tasks/events
	nextDepth := crawlTask.Depth + 1
	tasks := make([]models.CrawlTask, 0)
	outboxEvents := make([]models.OutboxEvent, 0)
	now := time.Now().UTC()

	// User-agent for robots.txt checking (matches the fetcher's user-agent)
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	// Capture the current parser span's trace context once for all child events.
	currentTraceCtx := telemetry.InjectTraceContext(ctx)

	for link := range linkSet {
		// Validate against scope rules
		if err := w.scopeValidator.Validate(link, nextDepth, jobConfig.Scopes); err != nil {
			w.logger.Debug("Link filtered by scope rules",
				zap.String("url", link),
				zap.Error(err),
			)
			continue
		}
		if len(allowedURLPatterns) > 0 {
			matched := false
			for _, pattern := range allowedURLPatterns {
				if pattern.MatchString(link) {
					matched = true
					break
				}
			}
			if !matched {
				w.logger.Debug("Link filtered by allowed_url_patterns",
					zap.String("url", link),
					zap.Strings("allowed_url_patterns", jobConfig.Scopes.AllowedURLPatterns),
				)
				continue
			}
		}

		// Check robots.txt rules
		allowed, err := w.robotsTxtService.IsAllowed(ctx, link, userAgent)
		if err != nil {
			w.logger.Debug("Failed to check robots.txt for link, allowing",
				zap.String("url", link),
				zap.Error(err),
			)
			// On error, proceed with enqueueing (permissive default)
		} else if !allowed {
			w.logger.Debug("Link disallowed by robots.txt",
				zap.String("url", link),
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

		// Create TaskEnqueuedEvent with the current parser span's trace context
		// so downstream fetch/parse spans are children of this parser span.
		event := events.NewTaskEnqueuedEvent(
			taskID.String(),
			crawlTask.JobID.String(),
			link,
			now,
		)
		event.TraceContext = currentTraceCtx

		// Marshal event to JSON
		payload, err := json.Marshal(event)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal event for task %s: %w", taskID.String(), err)
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
		return nil, nil, nil
	}

	w.logger.Debug("Prepared discovered links",
		zap.String("task_id", crawlTask.ID.String()),
		zap.Int("links_discovered", len(linkSet)),
		zap.Int("links_prepared", len(tasks)),
		zap.Uint64("next_depth", nextDepth),
	)

	return tasks, outboxEvents, nil
}
