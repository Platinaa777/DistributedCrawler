package crawljob

import (
	"context"

	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// FileTypeExportJSON represents the job-level JSON export
	FileTypeExportJSON = "json"
	// FileTypeExportCSV represents the job-level CSV export
	FileTypeExportCSV = "csv"
)

// GetJobExportFileURL generates a presigned URL for downloading a job export file
func (i *CrawlJobImplementation) GetJobExportFileURL(ctx context.Context, req *crawlergrpc.GetJobExportFileURLRequest) (*crawlergrpc.GetJobExportFileURLResponse, error) {
	job, err := i.crawlJobService.GetCrawlJob(ctx, service.GetCrawlJobQuery{
		ID: req.GetJobId(),
	})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "job not found: %v", err)
	}

	var objectKey string
	switch req.GetFileType() {
	case FileTypeExportJSON:
		if job.ExportJSONKey == nil || *job.ExportJSONKey == "" {
			return nil, status.Error(codes.NotFound, "JSON export not available for this job")
		}
		objectKey = *job.ExportJSONKey
	case FileTypeExportCSV:
		if job.ExportCSVKey == nil || *job.ExportCSVKey == "" {
			return nil, status.Error(codes.NotFound, "CSV export not available for this job")
		}
		objectKey = *job.ExportCSVKey
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid file_type: %s (must be 'json' or 'csv')", req.GetFileType())
	}

	url, err := i.urlGenerator.PresignGetURL(objectKey, URLExpirationMinutes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate presigned URL: %v", err)
	}

	return &crawlergrpc.GetJobExportFileURLResponse{
		Url:              url,
		ExpiresInSeconds: int32(URLExpirationMinutes * 60),
	}, nil
}
