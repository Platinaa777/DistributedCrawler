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

func (i *CrawlJobImplementation) ListTasksByJob(ctx context.Context, req *crawlergrpc.ListTasksByJobRequest) (*crawlergrpc.ListTasksByJobResponse, error) {
	query := service.ListTasksByJobQuery{
		JobID: req.JobId,
		Limit: int(req.Limit),
	}

	// Map sort field and direction (default: enqueued_at ASC)
	switch req.SortField {
	case crawlergrpc.TaskSortField_TASK_SORT_FIELD_URL:
		query.SortField = service.TaskSortByURL
		query.SortAsc = req.SortOrder != crawlergrpc.SortOrder_SORT_ORDER_DESC
	case crawlergrpc.TaskSortField_TASK_SORT_FIELD_STATUS:
		query.SortField = service.TaskSortByStatus
		query.SortAsc = req.SortOrder != crawlergrpc.SortOrder_SORT_ORDER_DESC
	case crawlergrpc.TaskSortField_TASK_SORT_FIELD_DEPTH:
		query.SortField = service.TaskSortByDepth
		query.SortAsc = req.SortOrder != crawlergrpc.SortOrder_SORT_ORDER_DESC
	default: // enqueued_at (default field, default ASC)
		query.SortField = service.TaskSortByEnqueuedAt
		query.SortAsc = req.SortOrder != crawlergrpc.SortOrder_SORT_ORDER_DESC
	}

	// Decode cursor if provided
	if req.Cursor != nil && *req.Cursor != "" {
		cursor, err := decodeTaskCursor(*req.Cursor)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid cursor: %v", err)
		}
		// Cursor carries its own sort info; use it for consistent pagination
		query.Cursor = cursor
		query.SortField = service.TaskSortField(cursor.SortField)
		query.SortAsc = cursor.SortAsc
	}

	// Map filters
	if req.Filter != nil {
		if req.Filter.Status != nil {
			query.Filter.Status = req.Filter.Status
		}
		if req.Filter.Url != nil {
			query.Filter.URL = req.Filter.Url
		}
		if req.Filter.MinDepth != nil {
			query.Filter.MinDepth = req.Filter.MinDepth
		}
		if req.Filter.MaxDepth != nil {
			query.Filter.MaxDepth = req.Filter.MaxDepth
		}
		if req.Filter.EnqueuedFrom != nil {
			t := req.Filter.EnqueuedFrom.AsTime()
			query.Filter.EnqueuedFrom = &t
		}
		if req.Filter.EnqueuedTo != nil {
			t := req.Filter.EnqueuedTo.AsTime()
			query.Filter.EnqueuedTo = &t
		}
	}

	result, err := i.crawlTaskService.ListTasksByJob(ctx, query)
	if err != nil {
		return nil, err
	}

	protoTasks := make([]*crawlergrpc.CrawlTask, 0, len(result.Tasks))
	for _, task := range result.Tasks {
		protoTasks = append(protoTasks, ToProtoCrawlTask(task))
	}

	response := &crawlergrpc.ListTasksByJobResponse{
		Tasks:   protoTasks,
		HasMore: result.HasMore,
	}

	// Encode next cursor if available
	if result.NextCursor != nil {
		encoded, err := encodeTaskCursor(result.NextCursor)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to encode cursor: %v", err)
		}
		response.NextCursor = encoded
	}

	return response, nil
}

// encodeTaskCursor converts a task cursor struct to a base64-encoded string
func encodeTaskCursor(cursor *service.ListTasksCursor) (string, error) {
	data, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(data), nil
}

// decodeTaskCursor converts a base64-encoded string back to a task cursor struct
func decodeTaskCursor(encoded string) (*service.ListTasksCursor, error) {
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	var cursor service.ListTasksCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, err
	}
	return &cursor, nil
}
