package crawljob

import (
	"context"
	"fmt"

	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	crawlergrpc "distributed-crawler/pkg/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// FileTypePages represents the pages bucket (HTML files)
	FileTypePages = "pages"
	// FileTypeResult represents the result bucket (JSON files)
	FileTypeResult = "result"
	// URLExpirationMinutes is the TTL for presigned URLs (10 minutes)
	URLExpirationMinutes = 10
)

// GetTaskFileURL generates a presigned URL for downloading a task file
func (i *CrawlJobImplementation) GetTaskFileURL(ctx context.Context, req *crawlergrpc.GetTaskFileURLRequest) (*crawlergrpc.GetTaskFileURLResponse, error) {
	// Get the task
	task, err := i.crawlTaskService.GetTask(ctx, service.GetCrawlTaskQuery{
		ID: req.GetTaskId(),
	})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "task not found: %v", err)
	}

	// Determine the object key based on file type
	var objectKey string
	switch req.GetFileType() {
	case FileTypePages:
		if task.MinioObjectKey == "" {
			return nil, status.Error(codes.NotFound, "HTML page file not available for this task")
		}
		objectKey = task.MinioObjectKey
	case FileTypeResult:
		if task.ResultObjectKey == nil || *task.ResultObjectKey == "" {
			return nil, status.Error(codes.NotFound, "result file not available for this task")
		}
		objectKey = *task.ResultObjectKey
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid file_type: %s (must be 'pages' or 'result')", req.GetFileType())
	}

	// Generate presigned URL
	url, err := i.urlGenerator.PresignGetURL(objectKey, URLExpirationMinutes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate presigned URL: %v", err)
	}

	return &crawlergrpc.GetTaskFileURLResponse{
		Url:              url,
		ExpiresInSeconds: int32(URLExpirationMinutes * 60),
	}, nil
}

// GetTaskFileURLByJobID generates a presigned URL with job context validation
func (i *CrawlJobImplementation) GetTaskFileURLByJobID(ctx context.Context, jobID valueobjects.CrawlJobID, taskID string, fileType string) (string, error) {
	// Get the task
	task, err := i.crawlTaskService.GetTask(ctx, service.GetCrawlTaskQuery{
		ID: taskID,
	})
	if err != nil {
		return "", fmt.Errorf("task not found: %w", err)
	}

	// Validate that the task belongs to the requested job
	if task.JobID != jobID {
		return "", fmt.Errorf("task %s does not belong to job %s", taskID, jobID)
	}

	// Determine the object key based on file type
	var objectKey string
	switch fileType {
	case FileTypePages:
		if task.MinioObjectKey == "" {
			return "", fmt.Errorf("HTML page file not available for this task")
		}
		objectKey = task.MinioObjectKey
	case FileTypeResult:
		if task.ResultObjectKey == nil || *task.ResultObjectKey == "" {
			return "", fmt.Errorf("result file not available for this task")
		}
		objectKey = *task.ResultObjectKey
	default:
		return "", fmt.Errorf("invalid file_type: %s", fileType)
	}

	// Generate presigned URL
	return i.urlGenerator.PresignGetURL(objectKey, URLExpirationMinutes)
}
