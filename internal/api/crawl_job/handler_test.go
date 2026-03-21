package crawljob

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/auth"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	crawlergrpc "distributed-crawler/pkg/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeCrawlJobService struct {
	createFn func(ctx context.Context, cmd service.CreateCrawlJobCommand) (valueobjects.CrawlJobID, error)
	getFn    func(ctx context.Context, query service.GetCrawlJobQuery) (*models.CrawlJob, error)
	listFn   func(ctx context.Context, query service.ListCrawlJobsQuery) (*service.ListCrawlJobsResult, error)
}

func (f fakeCrawlJobService) CreateCrawlJob(ctx context.Context, cmd service.CreateCrawlJobCommand) (valueobjects.CrawlJobID, error) {
	return f.createFn(ctx, cmd)
}
func (f fakeCrawlJobService) GetCrawlJob(ctx context.Context, query service.GetCrawlJobQuery) (*models.CrawlJob, error) {
	return f.getFn(ctx, query)
}
func (f fakeCrawlJobService) ListCrawlJobs(ctx context.Context, query service.ListCrawlJobsQuery) (*service.ListCrawlJobsResult, error) {
	return f.listFn(ctx, query)
}
func (f fakeCrawlJobService) DeleteCrawlJob(context.Context, service.DeleteCrawlJobCommand) error {
	return nil
}

type fakeCrawlTaskService struct {
	getFn       func(ctx context.Context, query service.GetCrawlTaskQuery) (*models.CrawlTask, error)
	listFn      func(ctx context.Context, query service.ListTasksByJobQuery) (*service.ListTasksResult, error)
	analyticsFn func(ctx context.Context, query service.GetTaskAnalyticsQuery) (*service.TaskAnalytics, error)
}

func (f fakeCrawlTaskService) GetTask(ctx context.Context, query service.GetCrawlTaskQuery) (*models.CrawlTask, error) {
	return f.getFn(ctx, query)
}
func (f fakeCrawlTaskService) ListTasksByJob(ctx context.Context, query service.ListTasksByJobQuery) (*service.ListTasksResult, error) {
	return f.listFn(ctx, query)
}
func (f fakeCrawlTaskService) GetTaskAnalytics(ctx context.Context, query service.GetTaskAnalyticsQuery) (*service.TaskAnalytics, error) {
	return f.analyticsFn(ctx, query)
}

type fakeURLGenerator struct {
	presignFn func(key string, ttlMinutes int) (string, error)
}

func (f fakeURLGenerator) PresignGetURL(key string, ttlMinutes int) (string, error) {
	return f.presignFn(key, ttlMinutes)
}

func TestCreateJob_MapsProtoConfigToCommand(t *testing.T) {
	t.Parallel()

	jobID := valueobjects.GenerateCrawlJobID()
	impl := NewImplementation(
		fakeCrawlJobService{
			createFn: func(ctx context.Context, cmd service.CreateCrawlJobCommand) (valueobjects.CrawlJobID, error) {
				assert.Equal(t, "Books", cmd.Config.Name)
				assert.Equal(t, "user-123", cmd.UserID)
				assert.Equal(t, models.JobTypeScheduled, cmd.Config.JobType)
				assert.Equal(t, models.CrawlModePaginationOnly, cmd.Config.CrawlMode)
				require.Len(t, cmd.Config.Seeds, 1)
				assert.Equal(t, "https://example.com", cmd.Config.Seeds[0].Url)
				return jobID, nil
			},
		},
		fakeCrawlTaskService{},
		fakeURLGenerator{},
	)

	ctx := context.WithValue(context.Background(), auth.UserIDContextKey, "user-123")
	resp, err := impl.CreateJob(ctx, &crawlergrpc.CreateJobRequest{
		Config: &crawlergrpc.CrawlJobConfig{
			Name:      "Books",
			JobType:   crawlergrpc.JobType_JOB_TYPE_SCHEDULED,
			CrawlMode: crawlergrpc.CrawlMode_CRAWL_MODE_PAGINATION_ONLY,
			Seeds:     []*crawlergrpc.Seed{{Url: "https://example.com"}},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, jobID.String(), resp.Id)
}

func TestListJobs_UsesDecodedCursorAndEncodesNextCursor(t *testing.T) {
	t.Parallel()

	createdAt := time.Now().UTC().Round(0)
	inputCursor := &service.ListCrawlJobsCursor{
		SortField: string(service.JobSortByStatus),
		SortAsc:   true,
		CreatedAt: createdAt,
		Status:    "COMPLETED",
		ID:        "job-1",
	}
	encodedInput, err := encodeCursor(inputCursor)
	require.NoError(t, err)

	nextCursor := &service.ListCrawlJobsCursor{
		SortField: string(service.JobSortByName),
		SortAsc:   false,
		CreatedAt: createdAt.Add(time.Minute),
		Name:      "next",
		ID:        "job-2",
	}
	jobID := valueobjects.GenerateCrawlJobID()
	configID := valueobjects.GenerateID()

	impl := NewImplementation(
		fakeCrawlJobService{
			listFn: func(ctx context.Context, query service.ListCrawlJobsQuery) (*service.ListCrawlJobsResult, error) {
				require.NotNil(t, query.Cursor)
				assert.Equal(t, inputCursor, query.Cursor)
				assert.Equal(t, service.JobSortByStatus, query.SortField)
				assert.True(t, query.SortAsc)
				require.NotNil(t, query.Filter.UserEmail)
				assert.Equal(t, "user@example.com", *query.Filter.UserEmail)
				return &service.ListCrawlJobsResult{
					Jobs: []*models.CrawlJob{{
						ID:           jobID,
						JobConfigID:  configID,
						Status:       models.TaskStatusCompleted,
						CreatedAt:    createdAt,
						ExportStatus: models.ExportStatusNotStarted,
					}},
					NextCursor: nextCursor,
					HasMore:    true,
				}, nil
			},
		},
		fakeCrawlTaskService{},
		fakeURLGenerator{},
	)

	resp, err := impl.ListJobs(context.Background(), &crawlergrpc.ListJobsRequest{
		Limit:  10,
		Cursor: &encodedInput,
		Filter: &crawlergrpc.JobListFilter{
			UserEmail: ptr("user@example.com"),
		},
	})
	require.NoError(t, err)
	assert.True(t, resp.HasMore)
	require.Len(t, resp.Jobs, 1)
	assert.Equal(t, jobID.String(), resp.Jobs[0].Id)

	decodedNext, err := decodeCursor(resp.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, nextCursor, decodedNext)
}

func TestListJobs_InvalidCursorReturnsInvalidArgument(t *testing.T) {
	t.Parallel()

	impl := NewImplementation(fakeCrawlJobService{}, fakeCrawlTaskService{}, fakeURLGenerator{})
	resp, err := impl.ListJobs(context.Background(), &crawlergrpc.ListJobsRequest{
		Cursor: ptr("%%%"),
	})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestListTasksByJob_MapsFiltersAndCursor(t *testing.T) {
	t.Parallel()

	enqueuedAt := time.Now().UTC().Round(0)
	depth := uint64(3)
	cursor := &service.ListTasksCursor{
		SortField:  string(service.TaskSortByDepth),
		SortAsc:    false,
		EnqueuedAt: enqueuedAt,
		Depth:      &depth,
		ID:         "task-1",
	}
	encodedCursor, err := encodeTaskCursor(cursor)
	require.NoError(t, err)

	jobID := valueobjects.GenerateCrawlJobID()
	taskID := valueobjects.GenerateCrawlTaskID()

	impl := NewImplementation(
		fakeCrawlJobService{},
		fakeCrawlTaskService{
			listFn: func(ctx context.Context, query service.ListTasksByJobQuery) (*service.ListTasksResult, error) {
				assert.Equal(t, "job-id", query.JobID)
				require.NotNil(t, query.Cursor)
				assert.Equal(t, cursor, query.Cursor)
				assert.Equal(t, service.TaskSortByDepth, query.SortField)
				assert.False(t, query.SortAsc)
				require.NotNil(t, query.Filter.URL)
				assert.Equal(t, "search", *query.Filter.URL)
				return &service.ListTasksResult{
					Tasks: []*models.CrawlTask{{
						ID:         taskID,
						JobID:      jobID,
						URL:        "https://example.com/item",
						Status:     models.TaskStatusParsed,
						EnqueuedAt: enqueuedAt,
						Depth:      depth,
					}},
					HasMore: false,
				}, nil
			},
		},
		fakeURLGenerator{},
	)

	resp, err := impl.ListTasksByJob(context.Background(), &crawlergrpc.ListTasksByJobRequest{
		JobId:  "job-id",
		Limit:  20,
		Cursor: &encodedCursor,
		Filter: &crawlergrpc.TaskListFilter{
			Url:      ptr("search"),
			MinDepth: ptrUint64(1),
			MaxDepth: ptrUint64(5),
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.Tasks, 1)
	assert.Equal(t, taskID.String(), resp.Tasks[0].Id)
}

func TestGetTaskAnalytics_ConvertsMaps(t *testing.T) {
	t.Parallel()

	impl := NewImplementation(
		fakeCrawlJobService{},
		fakeCrawlTaskService{
			analyticsFn: func(ctx context.Context, query service.GetTaskAnalyticsQuery) (*service.TaskAnalytics, error) {
				assert.Equal(t, "job-id", query.JobID)
				return &service.TaskAnalytics{
					StatusCounts: map[string]int64{"PARSED": 3},
					DepthCounts:  map[uint64]int64{0: 1, 1: 2},
					TotalCount:   3,
				}, nil
			},
		},
		fakeURLGenerator{},
	)

	resp, err := impl.GetTaskAnalytics(context.Background(), &crawlergrpc.GetTaskAnalyticsRequest{JobId: "job-id"})
	require.NoError(t, err)
	assert.Equal(t, int64(3), resp.Analytics.TotalCount)
	assert.Equal(t, int64(3), resp.Analytics.StatusCounts["PARSED"])
	assert.Equal(t, int64(2), resp.Analytics.DepthCounts[1])
}

func TestGetTaskFileURL_UsesRequestedObjectKey(t *testing.T) {
	t.Parallel()

	resultKey := "results/tasks/task-1.json"
	impl := NewImplementation(
		fakeCrawlJobService{},
		fakeCrawlTaskService{
			getFn: func(ctx context.Context, query service.GetCrawlTaskQuery) (*models.CrawlTask, error) {
				assert.Equal(t, "task-id", query.ID)
				return &models.CrawlTask{
					ID:              valueobjects.GenerateCrawlTaskID(),
					JobID:           valueobjects.GenerateCrawlJobID(),
					MinioObjectKey:  "pages/task-1.html",
					ResultObjectKey: &resultKey,
				}, nil
			},
		},
		fakeURLGenerator{
			presignFn: func(key string, ttlMinutes int) (string, error) {
				assert.Equal(t, resultKey, key)
				assert.Equal(t, URLExpirationMinutes, ttlMinutes)
				return "https://signed/result", nil
			},
		},
	)

	resp, err := impl.GetTaskFileURL(context.Background(), &crawlergrpc.GetTaskFileURLRequest{
		TaskId:   "task-id",
		FileType: FileTypeResult,
	})
	require.NoError(t, err)
	assert.Equal(t, "https://signed/result", resp.Url)
	assert.Equal(t, int32(600), resp.ExpiresInSeconds)
}

func TestGetTaskFileURL_ValidatesTypeAndAvailability(t *testing.T) {
	t.Parallel()

	impl := NewImplementation(
		fakeCrawlJobService{},
		fakeCrawlTaskService{
			getFn: func(ctx context.Context, query service.GetCrawlTaskQuery) (*models.CrawlTask, error) {
				return &models.CrawlTask{}, nil
			},
		},
		fakeURLGenerator{presignFn: func(key string, ttlMinutes int) (string, error) { return "", nil }},
	)

	resp, err := impl.GetTaskFileURL(context.Background(), &crawlergrpc.GetTaskFileURLRequest{
		TaskId:   "task-id",
		FileType: FileTypePages,
	})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, status.Code(err))

	resp, err = impl.GetTaskFileURL(context.Background(), &crawlergrpc.GetTaskFileURLRequest{
		TaskId:   "task-id",
		FileType: "bad",
	})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetTaskFileURLByJobID_RejectsTaskFromDifferentJob(t *testing.T) {
	t.Parallel()

	taskJobID := valueobjects.GenerateCrawlJobID()
	impl := NewImplementation(
		fakeCrawlJobService{},
		fakeCrawlTaskService{
			getFn: func(ctx context.Context, query service.GetCrawlTaskQuery) (*models.CrawlTask, error) {
				return &models.CrawlTask{JobID: taskJobID}, nil
			},
		},
		fakeURLGenerator{},
	)

	_, err := impl.GetTaskFileURLByJobID(context.Background(), valueobjects.GenerateCrawlJobID(), "task-id", FileTypePages)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not belong")
}

func TestGetTaskFileURLByJobID_ValidatesTypeAndAvailability(t *testing.T) {
	t.Parallel()

	jobID := valueobjects.GenerateCrawlJobID()
	resultKey := "results/task.json"
	impl := NewImplementation(
		fakeCrawlJobService{},
		fakeCrawlTaskService{
			getFn: func(ctx context.Context, query service.GetCrawlTaskQuery) (*models.CrawlTask, error) {
				return &models.CrawlTask{JobID: jobID, ResultObjectKey: &resultKey}, nil
			},
		},
		fakeURLGenerator{
			presignFn: func(key string, ttlMinutes int) (string, error) {
				return "https://signed/local", nil
			},
		},
	)

	url, err := impl.GetTaskFileURLByJobID(context.Background(), jobID, "task-id", FileTypeResult)
	require.NoError(t, err)
	assert.Equal(t, "https://signed/local", url)

	_, err = impl.GetTaskFileURLByJobID(context.Background(), jobID, "task-id", "bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid file_type")
}

func TestGetJobExportFileURL_ValidatesTypeAndUsesKey(t *testing.T) {
	t.Parallel()

	jsonKey := "exports/job-1.json"
	impl := NewImplementation(
		fakeCrawlJobService{
			getFn: func(ctx context.Context, query service.GetCrawlJobQuery) (*models.CrawlJob, error) {
				return &models.CrawlJob{
					ID:            valueobjects.GenerateCrawlJobID(),
					JobConfigID:   valueobjects.GenerateID(),
					CreatedAt:     time.Now().UTC(),
					ExportJSONKey: &jsonKey,
				}, nil
			},
		},
		fakeCrawlTaskService{},
		fakeURLGenerator{
			presignFn: func(key string, ttlMinutes int) (string, error) {
				assert.Equal(t, jsonKey, key)
				return "https://signed/export", nil
			},
		},
	)

	resp, err := impl.GetJobExportFileURL(context.Background(), &crawlergrpc.GetJobExportFileURLRequest{
		JobId:    "job-id",
		FileType: FileTypeExportJSON,
	})
	require.NoError(t, err)
	assert.Equal(t, "https://signed/export", resp.Url)

	resp, err = impl.GetJobExportFileURL(context.Background(), &crawlergrpc.GetJobExportFileURLRequest{
		JobId:    "job-id",
		FileType: "zip",
	})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestEncodeDecodeCursor_RoundTrip(t *testing.T) {
	t.Parallel()

	cursor := &service.ListCrawlJobsCursor{
		SortField: string(service.JobSortByCreatedAt),
		SortAsc:   true,
		CreatedAt: time.Now().UTC().Round(0),
		ID:        "job-1",
	}

	encoded, err := encodeCursor(cursor)
	require.NoError(t, err)
	decoded, err := decodeCursor(encoded)
	require.NoError(t, err)
	assert.Equal(t, cursor, decoded)

	raw, err := base64.URLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	assert.Equal(t, "job-1", payload["i"])
}

func ptr[T any](v T) *T { return &v }

func ptrUint64(v uint64) *uint64 { return &v }
