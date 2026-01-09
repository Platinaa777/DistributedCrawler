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

	"distributed-crawler/internal/domain/crawl/models"
	crawljob "distributed-crawler/internal/domain/crawl/repos/crawl_job"
	crawljobconfig "distributed-crawler/internal/domain/crawl/repos/crawl_job_config"
	crawltask "distributed-crawler/internal/domain/crawl/repos/crawl_task"
	"distributed-crawler/internal/domain/crawl/services"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/messaging/rabbitmq"

	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"
)

// ParserWorker consumes parsing tasks, loads HTML from MinIO, parses using DSL, and prints results
type ParserWorker struct {
	rmqClient     rabbitmq.Client
	parsingQueue  string
	contentStore  services.ContentStore
	taskRepo      crawltask.CrawlTaskRepository
	jobRepo       crawljob.CrawlJobRepository
	jobConfigRepo crawljobconfig.CrawlJobConfigRepository
	logger        *zap.Logger
}

// NewParserWorker creates a new parser worker
func NewParserWorker(
	rmqClient rabbitmq.Client,
	parsingQueue string,
	contentStore services.ContentStore,
	taskRepo crawltask.CrawlTaskRepository,
	jobRepo crawljob.CrawlJobRepository,
	jobConfigRepo crawljobconfig.CrawlJobConfigRepository,
	logger *zap.Logger,
) *ParserWorker {
	return &ParserWorker{
		rmqClient:     rmqClient,
		parsingQueue:  parsingQueue,
		contentStore:  contentStore,
		taskRepo:      taskRepo,
		jobRepo:       jobRepo,
		jobConfigRepo: jobConfigRepo,
		logger:        logger,
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

	// Print results to console
	w.printResults(task.TaskID, crawlTask.URL, result)

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
		return spec.Extractor.Default, err
	}

	// If value is nil/empty, return default
	if rawValue == nil || rawValue == "" {
		return spec.Extractor.Default, nil
	}

	// Apply transforms
	transformedValue := rawValue
	for _, transform := range spec.Transforms {
		transformedValue = w.applyTransform(transform, transformedValue)
	}

	// Convert to expected type
	finalValue, err := w.convertToType(transformedValue, spec.Type)
	if err != nil {
		return spec.Extractor.Default, err
	}

	return finalValue, nil
}

// applyExtractor extracts raw value from HTML based on selector type
func (w *ParserWorker) applyExtractor(
	spec models.ExtractorSpec,
	doc *goquery.Document,
	bodyText string,
	baseURL *url.URL,
) (any, error) {
	switch spec.Source {
	case models.SourceHTML:
		return w.extractFromHTML(spec, doc, baseURL)
	case models.SourceText:
		return w.extractFromText(spec, bodyText)
	case models.SourceFetchMeta, models.SourceResponseHeaders:
		// These would require fetch metadata to be passed in
		return spec.Default, nil
	default:
		return nil, fmt.Errorf("unsupported source type: %s", spec.Source)
	}
}

// extractFromHTML extracts data from HTML using various selectors
func (w *ParserWorker) extractFromHTML(
	spec models.ExtractorSpec,
	doc *goquery.Document,
	baseURL *url.URL,
) (any, error) {
	switch spec.SelectorType {
	case models.SelectorCSS:
		return w.extractWithCSS(spec, doc, baseURL)
	case models.SelectorMeta:
		return w.extractMetaTag(spec, doc)
	case models.SelectorURL:
		if baseURL != nil {
			return baseURL.String(), nil
		}
		return nil, fmt.Errorf("no URL available")
	default:
		return nil, fmt.Errorf("unsupported selector type: %s", spec.SelectorType)
	}
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
		if spec.Index != nil && *spec.Index < len(results) {
			return results[*spec.Index], nil
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
	if attribute != "" {
		value, exists := selection.Attr(attribute)
		if !exists {
			return ""
		}
		// Resolve relative URLs for href/src attributes
		if (attribute == "href" || attribute == "src") && baseURL != nil {
			if resolved, err := baseURL.Parse(value); err == nil {
				return resolved.String()
			}
		}
		return value
	}
	return strings.TrimSpace(selection.Text())
}

// extractMetaTag extracts content from meta tags
func (w *ParserWorker) extractMetaTag(spec models.ExtractorSpec, doc *goquery.Document) (any, error) {
	// Try name attribute
	selector := fmt.Sprintf("meta[name=%s]", spec.Selector)
	if content, exists := doc.Find(selector).First().Attr("content"); exists {
		return content, nil
	}

	// Try property attribute (for Open Graph tags)
	selector = fmt.Sprintf("meta[property=%s]", spec.Selector)
	if content, exists := doc.Find(selector).First().Attr("content"); exists {
		return content, nil
	}

	return nil, fmt.Errorf("meta tag not found: %s", spec.Selector)
}

// extractFromText extracts data from plain text using regex
func (w *ParserWorker) extractFromText(spec models.ExtractorSpec, bodyText string) (any, error) {
	if spec.SelectorType != models.SelectorRegex {
		return nil, fmt.Errorf("text source requires regex selector")
	}

	re, err := regexp.Compile(spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}

	if spec.Multiple {
		matches := re.FindAllString(bodyText, -1)
		if len(matches) == 0 {
			return nil, fmt.Errorf("no matches found")
		}

		if spec.Index != nil && *spec.Index < len(matches) {
			return matches[*spec.Index], nil
		}

		return matches, nil
	}

	match := re.FindString(bodyText)
	if match == "" {
		return nil, fmt.Errorf("no match found")
	}

	return match, nil
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
			if limit, ok := spec.Arg.(int); ok && limit < len(arr) {
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
		case int, int64:
			return v, nil
		case float64:
			return int64(v), nil
		case string:
			return strconv.ParseInt(v, 10, 64)
		}
	case models.ValueFloat:
		switch v := value.(type) {
		case float64:
			return v, nil
		case int, int64:
			return float64(v.(int64)), nil
		case string:
			return strconv.ParseFloat(v, 64)
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
		if value, ok := fields[spec.Input]; ok {
			if arr, ok := value.([]string); ok {
				return len(arr)
			}
		}
		return 0

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

// printResults prints extraction results to console
func (w *ParserWorker) printResults(taskID, url string, result *extractionResult) {
	output := map[string]any{
		"task_id": taskID,
		"url":     url,
		"fields":  result.Fields,
		"metrics": result.Metrics,
	}

	jsonOutput, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		w.logger.Error("Failed to marshal output", zap.Error(err))
		return
	}

	fmt.Println("\n" + string(jsonOutput) + "\n")
}
