package models

import (
	"testing"
	"time"
)

func TestCrawlTaskMarkAsFetchedClearsError(t *testing.T) {
	t.Parallel()

	task := CrawlTask{
		Status:       TaskStatusFailed,
		ErrorMessage: stringPtr("fetch failed"),
	}

	task.MarkAsFetched("https://example.com/final", "pages/job/task.html")

	if task.Status != TaskStatusFetched {
		t.Fatalf("expected status %q, got %q", TaskStatusFetched, task.Status)
	}
	if task.FinalURL == nil || *task.FinalURL != "https://example.com/final" {
		t.Fatalf("expected final URL to be set")
	}
	if task.MinioObjectKey != "pages/job/task.html" {
		t.Fatalf("expected minio object key to be set")
	}
	if task.ErrorMessage != nil {
		t.Fatalf("expected error message to be cleared")
	}
}

func TestCrawlTaskMarkAsParsedSetsResultMetadata(t *testing.T) {
	t.Parallel()

	task := CrawlTask{
		Status:       TaskStatusFailed,
		ErrorMessage: stringPtr("parse failed"),
	}
	parsedAt := time.Date(2026, time.February, 26, 12, 0, 0, 0, time.UTC)

	task.MarkAsParsed("results/tasks/task-1.json", "application/json", 128, parsedAt)

	if task.Status != TaskStatusParsed {
		t.Fatalf("expected status %q, got %q", TaskStatusParsed, task.Status)
	}
	if task.ResultObjectKey == nil || *task.ResultObjectKey != "results/tasks/task-1.json" {
		t.Fatalf("expected result object key to be set")
	}
	if task.ResultContentType == nil || *task.ResultContentType != "application/json" {
		t.Fatalf("expected result content type to be set")
	}
	if task.ResultSizeBytes == nil || *task.ResultSizeBytes != 128 {
		t.Fatalf("expected result size to be set")
	}
	if task.ResultCreatedAt == nil || !task.ResultCreatedAt.Equal(parsedAt) {
		t.Fatalf("expected result created at to be set")
	}
	if task.ErrorMessage != nil {
		t.Fatalf("expected error message to be cleared")
	}
}

func TestCrawlTaskMarkAsFailedClearsResultMetadata(t *testing.T) {
	t.Parallel()

	finalURL := "https://example.com/final"
	task := CrawlTask{
		FinalURL:          &finalURL,
		MinioObjectKey:    "pages/job/task.html",
		ResultObjectKey:   stringPtr("results/tasks/task-1.json"),
		ResultContentType: stringPtr("application/json"),
		ResultSizeBytes:   int64Ptr(128),
		ResultCreatedAt:   timePtr(time.Date(2026, time.February, 26, 12, 0, 0, 0, time.UTC)),
	}

	task.MarkAsFailed("failed to persist results")

	if task.Status != TaskStatusFailed {
		t.Fatalf("expected status %q, got %q", TaskStatusFailed, task.Status)
	}
	if task.ErrorMessage == nil || *task.ErrorMessage != "failed to persist results" {
		t.Fatalf("expected error message to be set")
	}
	if task.ResultObjectKey != nil || task.ResultContentType != nil || task.ResultSizeBytes != nil || task.ResultCreatedAt != nil {
		t.Fatalf("expected result metadata to be cleared")
	}
	if task.FinalURL == nil || *task.FinalURL != finalURL {
		t.Fatalf("expected fetch metadata to be preserved")
	}
	if task.MinioObjectKey != "pages/job/task.html" {
		t.Fatalf("expected minio object key to be preserved")
	}
}

func TestCrawlTaskMarkAsSkippedClearsResultMetadata(t *testing.T) {
	t.Parallel()

	task := CrawlTask{
		ResultObjectKey:   stringPtr("results/tasks/task-1.json"),
		ResultContentType: stringPtr("application/json"),
		ResultSizeBytes:   int64Ptr(128),
		ResultCreatedAt:   timePtr(time.Date(2026, time.February, 26, 12, 0, 0, 0, time.UTC)),
	}

	task.MarkAsSkipped("duplicate URL: https://example.com")

	if task.Status != TaskStatusSkipped {
		t.Fatalf("expected status %q, got %q", TaskStatusSkipped, task.Status)
	}
	if task.ErrorMessage == nil || *task.ErrorMessage != "duplicate URL: https://example.com" {
		t.Fatalf("expected error message to be set")
	}
	if task.ResultObjectKey != nil || task.ResultContentType != nil || task.ResultSizeBytes != nil || task.ResultCreatedAt != nil {
		t.Fatalf("expected result metadata to be cleared")
	}
}

func stringPtr(v string) *string {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}

func timePtr(v time.Time) *time.Time {
	return &v
}
