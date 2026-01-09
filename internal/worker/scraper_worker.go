package worker

import (
	"context"
	"distributed-crawler/internal/infra/messaging/rabbitmq"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"go.uber.org/zap"
)

// ScraperWorker is a worker that consumes crawl tasks from RabbitMQ and scrapes web pages using Colly
type ScraperWorker struct {
	rmqClient rabbitmq.Client
	queueName string
	logger    *zap.Logger
}

// NewScraperWorker creates a new scraper worker
func NewScraperWorker(
	rmqClient rabbitmq.Client,
	queueName string,
	logger *zap.Logger,
) *ScraperWorker {
	return &ScraperWorker{
		rmqClient: rmqClient,
		queueName: queueName,
		logger:    logger,
	}
}

// Start starts the worker and begins consuming messages from RabbitMQ
func (w *ScraperWorker) Start(ctx context.Context) error {
	w.logger.Info("Starting scraper worker", zap.String("queue", w.queueName))

	// Start consuming messages
	return w.rmqClient.Consume(ctx, w.queueName, w.handleMessage)
}

// handleMessage processes a single message from RabbitMQ
func (w *ScraperWorker) handleMessage(body []byte) error {
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

	// Scrape the page
	if err := w.scrapePage(task); err != nil {
		w.logger.Error("Failed to scrape page",
			zap.String("task_id", task.TaskID),
			zap.String("url", task.URL),
			zap.Error(err),
		)
		return err
	}

	w.logger.Info("Successfully scraped page",
		zap.String("task_id", task.TaskID),
		zap.String("url", task.URL),
	)

	return nil
}

// scrapePage scrapes a web page using Colly
func (w *ScraperWorker) scrapePage(task rabbitmq.CrawlTaskMessage) error {
	startTime := time.Now()

	// Create a new Colly collector
	c := colly.NewCollector(
		colly.AllowedDomains(), // Allow all domains
		colly.UserAgent("DistributedCrawler/1.0"),
	)

	// Set request timeout
	c.SetRequestTimeout(30 * time.Second)

	// Results to collect
	var results = struct {
		Title       string
		Description string
		Links       []string
		Images      []string
		StatusCode  int
		Headers     map[string]string
	}{
		Links:   make([]string, 0),
		Images:  make([]string, 0),
		Headers: make(map[string]string),
	}

	// On HTML element callbacks
	c.OnHTML("title", func(e *colly.HTMLElement) {
		results.Title = e.Text
	})

	c.OnHTML("meta[name=description]", func(e *colly.HTMLElement) {
		results.Description = e.Attr("content")
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("href"))
		if link != "" {
			results.Links = append(results.Links, link)
		}
	})

	c.OnHTML("img[src]", func(e *colly.HTMLElement) {
		imgSrc := e.Request.AbsoluteURL(e.Attr("src"))
		if imgSrc != "" {
			results.Images = append(results.Images, imgSrc)
		}
	})

	// Before request
	c.OnRequest(func(r *colly.Request) {
		w.logger.Debug("Visiting", zap.String("url", r.URL.String()))
	})

	// On response
	c.OnResponse(func(r *colly.Response) {
		results.StatusCode = r.StatusCode
		if r.Headers != nil {
			for key, values := range *r.Headers {
				if len(values) > 0 {
					results.Headers[key] = values[0]
				}
			}
		}
	})

	// On error
	c.OnError(func(r *colly.Response, err error) {
		w.logger.Error("Request failed",
			zap.String("url", r.Request.URL.String()),
			zap.Int("status", r.StatusCode),
			zap.Error(err),
		)
	})

	// Visit the URL
	if err := c.Visit(task.URL); err != nil {
		return fmt.Errorf("failed to visit URL: %w", err)
	}

	duration := time.Since(startTime)

	// Print results to console
	separator := strings.Repeat("=", 80)
	divider := strings.Repeat("-", 80)

	fmt.Println("\n" + separator)
	fmt.Printf("CRAWL RESULTS\n")
	fmt.Println(separator)
	fmt.Printf("Task ID:     %s\n", task.TaskID)
	fmt.Printf("Job ID:      %s\n", task.JobID)
	fmt.Printf("URL:         %s\n", task.URL)
	fmt.Printf("Status Code: %d\n", results.StatusCode)
	fmt.Printf("Duration:    %v\n", duration)
	fmt.Println(divider)
	fmt.Printf("Title:       %s\n", results.Title)
	fmt.Printf("Description: %s\n", results.Description)
	fmt.Println(divider)
	fmt.Printf("Links found: %d\n", len(results.Links))
	for i, link := range results.Links {
		if i >= 10 { // Limit output to first 10 links
			fmt.Printf("... and %d more links\n", len(results.Links)-10)
			break
		}
		fmt.Printf("  - %s\n", link)
	}
	fmt.Println(divider)
	fmt.Printf("Images found: %d\n", len(results.Images))
	for i, img := range results.Images {
		if i >= 5 { // Limit output to first 5 images
			fmt.Printf("... and %d more images\n", len(results.Images)-5)
			break
		}
		fmt.Printf("  - %s\n", img)
	}
	fmt.Println(divider)
	fmt.Printf("Headers:\n")
	for key, value := range results.Headers {
		fmt.Printf("  %s: %s\n", key, value)
	}
	fmt.Println(separator + "\n")

	return nil
}
