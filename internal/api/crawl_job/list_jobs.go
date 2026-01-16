package crawljob

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"distributed-crawler/internal/application/service"
	crawlergrpc "distributed-crawler/pkg/v1"
)

func (i *CrawlJobImplementation) ListJobs(ctx context.Context, req *crawlergrpc.ListJobsRequest) (*crawlergrpc.ListJobsResponse, error) {
	query := service.ListCrawlJobsQuery{
		Limit: int(req.Limit),
	}

	// Decode cursor if provided
	if req.Cursor != nil && *req.Cursor != "" {
		cursor, err := decodeCursor(*req.Cursor)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid cursor: %v", err)
		}
		query.Cursor = cursor
	}

	// Map filters
	if req.Filter != nil {
		if req.Filter.Name != nil {
			query.Filter.Name = req.Filter.Name
		}
		if req.Filter.Status != nil {
			query.Filter.Status = req.Filter.Status
		}
		if req.Filter.CreatedFrom != nil {
			t := req.Filter.CreatedFrom.AsTime()
			query.Filter.CreatedFrom = &t
		}
		if req.Filter.CreatedTo != nil {
			t := req.Filter.CreatedTo.AsTime()
			query.Filter.CreatedTo = &t
		}
	}

	result, err := i.crawlJobService.ListCrawlJobs(ctx, query)
	if err != nil {
		return nil, err
	}

	protoJobs := make([]*crawlergrpc.CrawlJob, 0, len(result.Jobs))
	for _, job := range result.Jobs {
		protoJobs = append(protoJobs, ToProtoCrawlJob(job))
	}

	response := &crawlergrpc.ListJobsResponse{
		Jobs:    protoJobs,
		HasMore: result.HasMore,
	}

	// Encode next cursor if available
	if result.NextCursor != nil {
		encoded, err := encodeCursor(result.NextCursor)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to encode cursor: %v", err)
		}
		response.NextCursor = encoded
	}

	return response, nil
}

// encodeCursor converts a cursor struct to a base64-encoded string
func encodeCursor(cursor *service.ListCrawlJobsCursor) (string, error) {
	data, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(data), nil
}

// decodeCursor converts a base64-encoded string back to a cursor struct
func decodeCursor(encoded string) (*service.ListCrawlJobsCursor, error) {
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	var cursor service.ListCrawlJobsCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, err
	}
	return &cursor, nil
}
