package worker

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"distributed-crawler/internal/domain/crawl/models"
	crawljob "distributed-crawler/internal/domain/crawl/repos/crawl_job"
	crawltask "distributed-crawler/internal/domain/crawl/repos/crawl_task"
	"distributed-crawler/internal/domain/crawl/services"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// ExportWorker aggregates crawl results and generates export files (Part B - ExportWorker)
type ExportWorker struct {
	jobRepo      crawljob.CrawlJobRepository
	taskRepo     crawltask.CrawlTaskRepository
	contentStore services.ContentStore
	txManager    persistence.TxManager
	pollInterval time.Duration
	batchSize    int
	logger       *zap.Logger
	tracer       trace.Tracer
	activeTasks  atomic.Int64
	accepting    atomic.Bool
}

// NewExportWorker creates a new export worker
func NewExportWorker(
	jobRepo crawljob.CrawlJobRepository,
	taskRepo crawltask.CrawlTaskRepository,
	contentStore services.ContentStore,
	txManager persistence.TxManager,
	pollInterval time.Duration,
	batchSize int,
	logger *zap.Logger,
	tracer trace.Tracer,
) *ExportWorker {
	worker := &ExportWorker{
		jobRepo:      jobRepo,
		taskRepo:     taskRepo,
		contentStore: contentStore,
		txManager:    txManager,
		pollInterval: pollInterval,
		batchSize:    batchSize,
		logger:       logger,
		tracer:       tracer,
	}
	worker.accepting.Store(true)
	return worker
}

// Start starts the export worker polling loop
func (w *ExportWorker) Start(ctx context.Context) error {
	w.logger.Info("Starting export worker",
		zap.Duration("poll_interval", w.pollInterval),
		zap.Int("batch_size", w.batchSize),
	)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Process immediately on startup
	if !w.accepting.Load() {
		w.logger.Info("Export worker drain requested before start")
		return nil
	}
	w.processEligibleJobs(ctx)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Export worker stopped")
			return ctx.Err()
		case <-ticker.C:
			if !w.accepting.Load() {
				w.logger.Info("Export worker drain completed")
				return nil
			}
			w.processEligibleJobs(ctx)
		}
	}
}

// ActiveTasks returns the number of jobs currently being exported.
func (w *ExportWorker) ActiveTasks() int32 {
	return int32(w.activeTasks.Load())
}

// StopAccepting prevents new export batches from starting.
func (w *ExportWorker) StopAccepting() {
	w.accepting.Store(false)
}

// processEligibleJobs finds and processes jobs that need exporting
func (w *ExportWorker) processEligibleJobs(ctx context.Context) {
	err := w.txManager.ReadCommitted(ctx, func(ctxTX context.Context) error {
		jobs, err := w.jobRepo.ListEligibleForExport(ctxTX, w.batchSize)
		if err != nil {
			return fmt.Errorf("failed to list eligible jobs for export: %w", err)
		}

		if len(jobs) == 0 {
			w.logger.Debug("No jobs eligible for export")
			return nil
		}

		w.logger.Info("Found jobs eligible for export", zap.Int("count", len(jobs)))

		for _, job := range jobs {
			if err := w.processJob(ctxTX, job); err != nil {
				w.logger.Error("Failed to process job export",
					zap.String("job_id", job.ID.String()),
					zap.Error(err),
				)
			}
		}

		return nil
	})
	if err != nil {
		w.logger.Error("Failed to process export batch", zap.Error(err))
	}
}

// processJob processes export for a single job
func (w *ExportWorker) processJob(ctx context.Context, job *models.CrawlJob) error {
	w.activeTasks.Add(1)
	defer w.activeTasks.Add(-1)

	if w.tracer != nil {
		var span trace.Span
		ctx, span = w.tracer.Start(ctx, "export_job",
			trace.WithAttributes(
				attribute.String("job.id", job.ID.String()),
			),
		)
		defer func() {
			span.End()
		}()
	}

	w.logger.Info("Starting export for job", zap.String("job_id", job.ID.String()))

	// Load all tasks for the job
	tasks, err := w.taskRepo.ListByJob(ctx, job.ID)
	if err != nil {
		w.failExport(ctx, job.ID, fmt.Sprintf("failed to load tasks: %v", err))
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	w.logger.Info("Loaded tasks for job",
		zap.String("job_id", job.ID.String()),
		zap.Int("task_count", len(tasks)),
	)

	// Load results for completed tasks
	results, err := w.loadTaskResults(ctx, tasks)
	if err != nil {
		w.failExport(ctx, job.ID, fmt.Sprintf("failed to load task results: %v", err))
		return fmt.Errorf("failed to load task results: %w", err)
	}

	w.logger.Info("Loaded task results",
		zap.String("job_id", job.ID.String()),
		zap.Int("result_count", len(results)),
	)

	// Generate JSON report
	jsonKey, err := w.generateJSONReport(ctx, job.ID, results)
	if err != nil {
		w.failExport(ctx, job.ID, fmt.Sprintf("failed to generate JSON report: %v", err))
		return fmt.Errorf("failed to generate JSON report: %w", err)
	}

	// Generate CSV report
	csvKey, err := w.generateCSVReport(ctx, job.ID, results)
	if err != nil {
		w.failExport(ctx, job.ID, fmt.Sprintf("failed to generate CSV report: %v", err))
		return fmt.Errorf("failed to generate CSV report: %w", err)
	}

	// Mark export as completed
	exportedAt := time.Now().UTC()
	job.MarkAsExported(jsonKey, csvKey, exportedAt)
	if err := w.jobRepo.Update(ctx, *job); err != nil {
		return fmt.Errorf("failed to mark export as completed: %w", err)
	}

	// Set span success status
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(
			attribute.Int("export.task_count", len(tasks)),
			attribute.Int("export.result_count", len(results)),
			attribute.String("export.json_key", jsonKey),
			attribute.String("export.csv_key", csvKey),
		)
		span.SetStatus(codes.Ok, "")
	}

	w.logger.Info("Export completed successfully",
		zap.String("job_id", job.ID.String()),
		zap.String("json_key", jsonKey),
		zap.String("csv_key", csvKey),
	)

	return nil
}

// TaskResult represents a single task's extraction result
type TaskResult struct {
	TaskID  string         `json:"task_id"`
	URL     string         `json:"url"`
	Status  string         `json:"status"`
	Fields  map[string]any `json:"fields,omitempty"`
	Metrics map[string]any `json:"metrics,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// loadTaskResults loads result JSON files from S3 for all tasks
func (w *ExportWorker) loadTaskResults(ctx context.Context, tasks []*models.CrawlTask) ([]TaskResult, error) {
	results := make([]TaskResult, 0, len(tasks))

	for _, task := range tasks {
		result := TaskResult{
			TaskID: task.ID.String(),
			URL:    task.URL,
			Status: task.Status.String(),
		}

		// Only load result JSON for completed tasks with result_object_key
		if (task.Status == models.TaskStatusCompleted || task.Status == models.TaskStatusParsed) && task.ResultObjectKey != nil && *task.ResultObjectKey != "" {
			jsonData, err := w.contentStore.Get(ctx, *task.ResultObjectKey)
			if err != nil {
				w.logger.Warn("Failed to load task result from S3",
					zap.String("task_id", task.ID.String()),
					zap.String("result_key", *task.ResultObjectKey),
					zap.Error(err),
				)
				result.Error = fmt.Sprintf("failed to load result: %v", err)
			} else {
				// Parse JSON to extract fields and metrics
				var taskOutput map[string]any
				if err := json.Unmarshal(jsonData, &taskOutput); err != nil {
					w.logger.Warn("Failed to parse task result JSON",
						zap.String("task_id", task.ID.String()),
						zap.Error(err),
					)
					result.Error = fmt.Sprintf("failed to parse JSON: %v", err)
				} else {
					// Extract fields and metrics from the output
					if fields, ok := taskOutput["fields"].(map[string]any); ok {
						result.Fields = fields
					}
					if metrics, ok := taskOutput["metrics"].(map[string]any); ok {
						result.Metrics = metrics
					}
				}
			}
		} else if task.Status == models.TaskStatusFailed {
			result.Error = "task failed during crawl/parse"
		}

		results = append(results, result)
	}

	return results, nil
}

// generateJSONReport creates a JSON report and uploads to S3
func (w *ExportWorker) generateJSONReport(ctx context.Context, jobID valueobjects.CrawlJobID, results []TaskResult) (string, error) {
	// Build report structure
	report := map[string]any{
		"job_id":      jobID.String(),
		"exported_at": time.Now().UTC().Format(time.RFC3339),
		"total_tasks": len(results),
		"results":     results,
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON report: %w", err)
	}

	// Determine S3 object key
	objectKey := fmt.Sprintf("exports/jobs/%s/report.json", jobID.String())

	// Upload to S3
	if err := w.contentStore.Store(ctx, objectKey, jsonData, "application/json"); err != nil {
		return "", fmt.Errorf("failed to upload JSON report to S3: %w", err)
	}

	w.logger.Info("JSON report uploaded to S3",
		zap.String("job_id", jobID.String()),
		zap.String("object_key", objectKey),
		zap.Int("size_bytes", len(jsonData)),
	)

	return objectKey, nil
}

// generateCSVReport creates a CSV report and uploads to S3
func (w *ExportWorker) generateCSVReport(ctx context.Context, jobID valueobjects.CrawlJobID, results []TaskResult) (string, error) {
	if len(results) == 0 {
		// Create empty CSV for jobs with no results
		objectKey := fmt.Sprintf("exports/jobs/%s/report.csv", jobID.String())
		emptyCSV := []byte("task_id,url,status\n")
		if err := w.contentStore.Store(ctx, objectKey, emptyCSV, "text/csv"); err != nil {
			return "", fmt.Errorf("failed to upload empty CSV: %w", err)
		}
		return objectKey, nil
	}

	// Compute union of all field names
	fieldNames := w.collectFieldNames(results)

	// Build CSV in memory
	var buf strings.Builder
	csvWriter := csv.NewWriter(&buf)

	// Write header
	header := []string{"task_id", "url", "status"}
	header = append(header, fieldNames...)
	header = append(header, "error")
	if err := csvWriter.Write(header); err != nil {
		return "", fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write rows
	for _, result := range results {
		row := []string{
			result.TaskID,
			result.URL,
			result.Status,
		}

		// Add field values (in order of fieldNames)
		for _, fieldName := range fieldNames {
			if result.Fields != nil {
				if val, ok := result.Fields[fieldName]; ok {
					row = append(row, fmt.Sprintf("%v", val))
				} else {
					row = append(row, "")
				}
			} else {
				row = append(row, "")
			}
		}

		// Add error column
		row = append(row, result.Error)

		if err := csvWriter.Write(row); err != nil {
			return "", fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return "", fmt.Errorf("CSV writer error: %w", err)
	}

	csvData := []byte(buf.String())

	// Determine S3 object key
	objectKey := fmt.Sprintf("exports/jobs/%s/report.csv", jobID.String())

	// Upload to S3
	if err := w.contentStore.Store(ctx, objectKey, csvData, "text/csv"); err != nil {
		return "", fmt.Errorf("failed to upload CSV report to S3: %w", err)
	}

	w.logger.Info("CSV report uploaded to S3",
		zap.String("job_id", jobID.String()),
		zap.String("object_key", objectKey),
		zap.Int("size_bytes", len(csvData)),
		zap.Int("field_count", len(fieldNames)),
	)

	return objectKey, nil
}

// collectFieldNames computes the union of all field names across results
func (w *ExportWorker) collectFieldNames(results []TaskResult) []string {
	fieldSet := make(map[string]bool)

	for _, result := range results {
		if result.Fields != nil {
			for fieldName := range result.Fields {
				fieldSet[fieldName] = true
			}
		}
	}

	// Convert to sorted slice for deterministic CSV column order
	fieldNames := make([]string, 0, len(fieldSet))
	for fieldName := range fieldSet {
		fieldNames = append(fieldNames, fieldName)
	}
	sort.Strings(fieldNames)

	return fieldNames
}

// failExport marks the export as failed with an error message
func (w *ExportWorker) failExport(ctx context.Context, jobID valueobjects.CrawlJobID, errorMsg string) {
	w.logger.Error("Export failed for job",
		zap.String("job_id", jobID.String()),
		zap.String("error", errorMsg),
	)

	if err := w.jobRepo.FailExport(ctx, jobID, errorMsg); err != nil {
		w.logger.Error("Failed to mark export as failed",
			zap.String("job_id", jobID.String()),
			zap.Error(err),
		)
	}
}
