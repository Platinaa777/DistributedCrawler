package worker

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/repos/page_extract"
	"distributed-crawler/internal/domain/crawl/repos/page_fetch"
	"distributed-crawler/internal/domain/crawl/services"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/messaging/rabbitmq"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"
)

// ParserWorker consumes parsing tasks, loads HTML from MinIO, parses, and saves results
type ParserWorker struct {
	rmqClient    rabbitmq.Client
	parsingQueue string
	contentStore services.ContentStore
	fetchRepo    page_fetch.PageFetchRepository
	extractRepo  page_extract.PageExtractRepository
	logger       *zap.Logger
}

// NewParserWorker creates a new parser worker
func NewParserWorker(
	rmqClient rabbitmq.Client,
	parsingQueue string,
	contentStore services.ContentStore,
	fetchRepo page_fetch.PageFetchRepository,
	extractRepo page_extract.PageExtractRepository,
	logger *zap.Logger,
) *ParserWorker {
	return &ParserWorker{
		rmqClient:    rmqClient,
		parsingQueue: parsingQueue,
		contentStore: contentStore,
		fetchRepo:    fetchRepo,
		extractRepo:  extractRepo,
		logger:       logger,
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

	// Load fetch metadata to get MinIO object key
	fetchMeta, err := w.fetchRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		w.logger.Error("Failed to load fetch metadata",
			zap.String("task_id", task.TaskID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to load fetch metadata: %w", err)
	}

	// Load HTML from MinIO
	htmlContent, err := w.contentStore.Get(ctx, fetchMeta.MinioObjectKey)
	if err != nil {
		w.logger.Error("Failed to load HTML from MinIO",
			zap.String("task_id", task.TaskID),
			zap.String("minio_key", fetchMeta.MinioObjectKey),
			zap.Error(err),
		)
		return fmt.Errorf("failed to load HTML from MinIO: %w", err)
	}

	// Parse HTML
	parseResult, err := w.parseHTML(ctx, taskID, fetchMeta.URL, htmlContent)
	if err != nil {
		w.logger.Error("Failed to parse HTML",
			zap.String("task_id", task.TaskID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Save results to database
	if err := w.saveResults(ctx, parseResult); err != nil {
		w.logger.Error("Failed to save parse results",
			zap.String("task_id", task.TaskID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save results: %w", err)
	}

	w.logger.Info("Successfully processed parsing task",
		zap.String("task_id", task.TaskID),
		zap.Int("link_count", parseResult.Extract.LinkCount),
		zap.Int("image_count", parseResult.Extract.ImageCount),
		zap.Int("word_count", parseResult.Extract.WordCount),
	)

	return nil
}

// parseResult contains the results of HTML parsing
type parseResult struct {
	Extract *models.PageExtract
	Links   []*models.PageLink
	Images  []*models.PageImage
}

// parseHTML parses HTML content and extracts structured data
func (w *ParserWorker) parseHTML(ctx context.Context, taskID valueobjects.CrawlTaskID, pageURL string, htmlContent []byte) (*parseResult, error) {
	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlContent)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Parse page URL for external link detection
	baseURL, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse page URL: %w", err)
	}

	// Extract title
	var title *string
	if titleText := doc.Find("title").First().Text(); titleText != "" {
		titleText = strings.TrimSpace(titleText)
		title = &titleText
	}

	// Extract meta description
	var metaDesc *string
	if desc, exists := doc.Find("meta[name=description]").First().Attr("content"); exists {
		desc = strings.TrimSpace(desc)
		metaDesc = &desc
	}

	// Extract canonical URL
	var canonical *string
	if canonURL, exists := doc.Find("link[rel=canonical]").First().Attr("href"); exists {
		canonURL = strings.TrimSpace(canonURL)
		canonical = &canonURL
	}

	// Extract links
	var links []*models.PageLink
	externalLinkCount := 0
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		// Resolve relative URLs
		linkURL, err := baseURL.Parse(href)
		if err != nil {
			return
		}
		absoluteURL := linkURL.String()

		// Get anchor text
		anchorText := strings.TrimSpace(s.Text())
		var anchorPtr *string
		if anchorText != "" {
			anchorPtr = &anchorText
		}

		// Check if external
		isExternal := linkURL.Host != baseURL.Host

		if isExternal {
			externalLinkCount++
		}

		links = append(links, &models.PageLink{
			ID:         valueobjects.GeneratePageLinkID(),
			TaskID:     taskID,
			URL:        absoluteURL,
			AnchorText: anchorPtr,
			IsExternal: isExternal,
			CreatedAt:  time.Now(),
		})
	})

	// Extract images
	var images []*models.PageImage
	doc.Find("img[src]").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists || src == "" {
			return
		}

		// Resolve relative URLs
		imgURL, err := baseURL.Parse(src)
		if err != nil {
			return
		}
		absoluteURL := imgURL.String()

		// Get alt text
		altText, _ := s.Attr("alt")
		var altPtr *string
		if altText != "" {
			altText = strings.TrimSpace(altText)
			altPtr = &altText
		}

		images = append(images, &models.PageImage{
			ID:        valueobjects.GeneratePageImageID(),
			TaskID:    taskID,
			URL:       absoluteURL,
			AltText:   altPtr,
			CreatedAt: time.Now(),
		})
	})

	// Compute word count (simple approximation)
	bodyText := doc.Find("body").Text()
	words := strings.Fields(bodyText)
	wordCount := len(words)

	// Build page extract
	extract := &models.PageExtract{
		TaskID:            taskID,
		Title:             title,
		MetaDescription:   metaDesc,
		CanonicalURL:      canonical,
		Metadata:          nil, // Can be extended for custom extraction
		LinkCount:         len(links),
		ImageCount:        len(images),
		ExternalLinkCount: externalLinkCount,
		WordCount:         wordCount,
		ParsedAt:          time.Now(),
		CreatedAt:         time.Now(),
	}

	return &parseResult{
		Extract: extract,
		Links:   links,
		Images:  images,
	}, nil
}

// saveResults saves all parse results to database
func (w *ParserWorker) saveResults(ctx context.Context, result *parseResult) error {
	// Save page extract (UPSERT)
	if err := w.extractRepo.Save(ctx, result.Extract); err != nil {
		return fmt.Errorf("failed to save page extract: %w", err)
	}

	// Save links (bulk insert with ON CONFLICT DO NOTHING)
	if len(result.Links) > 0 {
		if err := w.extractRepo.SaveLinks(ctx, result.Links); err != nil {
			return fmt.Errorf("failed to save links: %w", err)
		}
	}

	// Save images (bulk insert with ON CONFLICT DO NOTHING)
	if len(result.Images) > 0 {
		if err := w.extractRepo.SaveImages(ctx, result.Images); err != nil {
			return fmt.Errorf("failed to save images: %w", err)
		}
	}

	return nil
}
