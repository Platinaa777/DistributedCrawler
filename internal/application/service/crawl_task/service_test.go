package crawltask

import (
	"context"
	"errors"
	"testing"
	"time"

	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
	crawltaskrepo "distributed-crawler/internal/domain/crawl/repos/crawl_task"
	"distributed-crawler/internal/domain/crawl/valueobjects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type crawlTaskRepoFake struct {
	getFn       func(ctx context.Context, id valueobjects.CrawlTaskID) (*models.CrawlTask, error)
	listFn      func(ctx context.Context, query service.ListTasksByJobQuery) (*service.ListTasksResult, error)
	analyticsFn func(ctx context.Context, jobID valueobjects.CrawlJobID) (*service.TaskAnalytics, error)
}

func (f crawlTaskRepoFake) Create(context.Context, models.CrawlTask) (valueobjects.CrawlTaskID, error) {
	return valueobjects.CrawlTaskID{}, nil
}
func (f crawlTaskRepoFake) BulkCreate(context.Context, []models.CrawlTask) ([]valueobjects.CrawlTaskID, error) {
	return nil, nil
}
func (f crawlTaskRepoFake) Get(ctx context.Context, id valueobjects.CrawlTaskID) (*models.CrawlTask, error) {
	return f.getFn(ctx, id)
}
func (f crawlTaskRepoFake) Update(context.Context, models.CrawlTask) error { return nil }
func (f crawlTaskRepoFake) ListByJob(context.Context, valueobjects.CrawlJobID) ([]*models.CrawlTask, error) {
	return nil, nil
}
func (f crawlTaskRepoFake) ListByStatus(context.Context, models.TaskStatus, int) ([]*models.CrawlTask, error) {
	return nil, nil
}
func (f crawlTaskRepoFake) ListWithCursor(ctx context.Context, query service.ListTasksByJobQuery) (*service.ListTasksResult, error) {
	return f.listFn(ctx, query)
}
func (f crawlTaskRepoFake) GetAnalyticsByJob(ctx context.Context, jobID valueobjects.CrawlJobID) (*service.TaskAnalytics, error) {
	return f.analyticsFn(ctx, jobID)
}
func (f crawlTaskRepoFake) SetTaskResult(context.Context, valueobjects.CrawlTaskID, string, string, int64) error {
	return nil
}
func (f crawlTaskRepoFake) ExistsByJobIDAndURL(context.Context, valueobjects.CrawlJobID, string) (bool, error) {
	return false, nil
}
func (f crawlTaskRepoFake) ListStaleInProgress(context.Context, time.Time, int) ([]*models.CrawlTask, error) {
	return nil, nil
}

var _ crawltaskrepo.CrawlTaskRepository = crawlTaskRepoFake{}

func TestGetTask_ValidatesIDAndWrapsRepoError(t *testing.T) {
	t.Parallel()

	svc := NewCrawlTaskService(crawlTaskRepoFake{
		getFn: func(ctx context.Context, id valueobjects.CrawlTaskID) (*models.CrawlTask, error) {
			return nil, errors.New("db")
		},
		listFn:      func(ctx context.Context, query service.ListTasksByJobQuery) (*service.ListTasksResult, error) { return nil, nil },
		analyticsFn: func(ctx context.Context, jobID valueobjects.CrawlJobID) (*service.TaskAnalytics, error) { return nil, nil },
	})

	task, err := svc.GetTask(context.Background(), service.GetCrawlTaskQuery{ID: "bad"})
	require.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "invalid task ID")

	taskID := valueobjects.GenerateCrawlTaskID()
	task, err = svc.GetTask(context.Background(), service.GetCrawlTaskQuery{ID: taskID.String()})
	require.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "failed to get crawl task")
}

func TestListTasksByJob_AppliesDefaultsAndValidation(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(0)
	svc := NewCrawlTaskService(crawlTaskRepoFake{
		getFn: func(ctx context.Context, id valueobjects.CrawlTaskID) (*models.CrawlTask, error) { return nil, nil },
		listFn: func(ctx context.Context, query service.ListTasksByJobQuery) (*service.ListTasksResult, error) {
			assert.Equal(t, 20, query.Limit)
			assert.Equal(t, "job-id", query.JobID)
			return &service.ListTasksResult{
				Tasks: []*models.CrawlTask{{
					ID:         valueobjects.GenerateCrawlTaskID(),
					JobID:      valueobjects.GenerateCrawlJobID(),
					URL:        "https://example.com",
					Status:     models.TaskStatusParsed,
					EnqueuedAt: now,
				}},
			}, nil
		},
		analyticsFn: func(ctx context.Context, jobID valueobjects.CrawlJobID) (*service.TaskAnalytics, error) { return nil, nil },
	})

	result, err := svc.ListTasksByJob(context.Background(), service.ListTasksByJobQuery{JobID: "job-id"})
	require.NoError(t, err)
	require.Len(t, result.Tasks, 1)

	status := "BAD"
	_, err = svc.ListTasksByJob(context.Background(), service.ListTasksByJobQuery{
		Filter: service.ListTasksFilter{Status: &status},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")

	minDepth, maxDepth := uint64(5), uint64(1)
	_, err = svc.ListTasksByJob(context.Background(), service.ListTasksByJobQuery{
		Filter: service.ListTasksFilter{MinDepth: &minDepth, MaxDepth: &maxDepth},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "min_depth")

	from, to := now, now.Add(-time.Hour)
	_, err = svc.ListTasksByJob(context.Background(), service.ListTasksByJobQuery{
		Filter: service.ListTasksFilter{EnqueuedFrom: &from, EnqueuedTo: &to},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "enqueued_from")
}

func TestListTasksByJob_CapsLimitAndWrapsRepoError(t *testing.T) {
	t.Parallel()

	svc := NewCrawlTaskService(crawlTaskRepoFake{
		getFn: func(ctx context.Context, id valueobjects.CrawlTaskID) (*models.CrawlTask, error) { return nil, nil },
		listFn: func(ctx context.Context, query service.ListTasksByJobQuery) (*service.ListTasksResult, error) {
			assert.Equal(t, 100, query.Limit)
			return nil, errors.New("repo fail")
		},
		analyticsFn: func(ctx context.Context, jobID valueobjects.CrawlJobID) (*service.TaskAnalytics, error) { return nil, nil },
	})

	result, err := svc.ListTasksByJob(context.Background(), service.ListTasksByJobQuery{Limit: 101})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to list tasks")
}

func TestGetTaskAnalytics_ValidatesIDAndWrapsRepoError(t *testing.T) {
	t.Parallel()

	jobID := valueobjects.GenerateCrawlJobID()
	svc := NewCrawlTaskService(crawlTaskRepoFake{
		getFn:  func(ctx context.Context, id valueobjects.CrawlTaskID) (*models.CrawlTask, error) { return nil, nil },
		listFn: func(ctx context.Context, query service.ListTasksByJobQuery) (*service.ListTasksResult, error) { return nil, nil },
		analyticsFn: func(ctx context.Context, gotJobID valueobjects.CrawlJobID) (*service.TaskAnalytics, error) {
			assert.Equal(t, jobID, gotJobID)
			return nil, errors.New("repo")
		},
	})

	analytics, err := svc.GetTaskAnalytics(context.Background(), service.GetTaskAnalyticsQuery{JobID: "bad"})
	require.Error(t, err)
	assert.Nil(t, analytics)
	assert.Contains(t, err.Error(), "invalid job ID")

	analytics, err = svc.GetTaskAnalytics(context.Background(), service.GetTaskAnalyticsQuery{JobID: jobID.String()})
	require.Error(t, err)
	assert.Nil(t, analytics)
	assert.Contains(t, err.Error(), "failed to get task analytics")
}

