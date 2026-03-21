package crawljob

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/events"
	"distributed-crawler/internal/domain/crawl/models"
	crawljobrepo "distributed-crawler/internal/domain/crawl/repos/crawl_job"
	crawljobconfigrepo "distributed-crawler/internal/domain/crawl/repos/crawl_job_config"
	crawltaskrepo "distributed-crawler/internal/domain/crawl/repos/crawl_task"
	outboxrepo "distributed-crawler/internal/domain/crawl/repos/outbox"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type crawlJobRepoFake struct {
	createFn func(ctx context.Context, entity models.CrawlJob) (valueobjects.CrawlJobID, error)
	getFn    func(ctx context.Context, id valueobjects.CrawlJobID) (*models.CrawlJob, error)
	listFn   func(ctx context.Context, query service.ListCrawlJobsQuery) (*service.ListCrawlJobsResult, error)
}

func (f crawlJobRepoFake) Create(ctx context.Context, entity models.CrawlJob) (valueobjects.CrawlJobID, error) {
	return f.createFn(ctx, entity)
}
func (f crawlJobRepoFake) Get(ctx context.Context, id valueobjects.CrawlJobID) (*models.CrawlJob, error) {
	return f.getFn(ctx, id)
}
func (f crawlJobRepoFake) Update(context.Context, models.CrawlJob) error { return nil }
func (f crawlJobRepoFake) ListWithCursor(ctx context.Context, query service.ListCrawlJobsQuery) (*service.ListCrawlJobsResult, error) {
	return f.listFn(ctx, query)
}
func (f crawlJobRepoFake) List(context.Context, models.TaskStatus, int, int) ([]*models.CrawlJob, error) {
	return nil, nil
}
func (f crawlJobRepoFake) ListAll(context.Context, int, int) ([]*models.CrawlJob, error) { return nil, nil }
func (f crawlJobRepoFake) GetLatestByConfigID(context.Context, valueobjects.ID) (*models.CrawlJob, error) {
	return nil, nil
}
func (f crawlJobRepoFake) ListEligibleForExport(context.Context, int) ([]*models.CrawlJob, error) { return nil, nil }
func (f crawlJobRepoFake) TryStartExport(context.Context, valueobjects.CrawlJobID) (bool, error)   { return false, nil }
func (f crawlJobRepoFake) FailExport(context.Context, valueobjects.CrawlJobID, string) error        { return nil }

type crawlJobConfigRepoFake struct {
	createFn func(ctx context.Context, entity models.CrawlJobConfig) (valueobjects.ID, error)
	getFn    func(ctx context.Context, id valueobjects.ID) (*models.CrawlJobConfig, error)
}

func (f crawlJobConfigRepoFake) Create(ctx context.Context, entity models.CrawlJobConfig) (valueobjects.ID, error) {
	return f.createFn(ctx, entity)
}
func (f crawlJobConfigRepoFake) Get(ctx context.Context, id valueobjects.ID) (*models.CrawlJobConfig, error) {
	return f.getFn(ctx, id)
}
func (f crawlJobConfigRepoFake) Update(context.Context, models.CrawlJobConfig) error              { return nil }
func (f crawlJobConfigRepoFake) Delete(context.Context, valueobjects.ID) error                     { return nil }
func (f crawlJobConfigRepoFake) ListAllScheduled(context.Context, int, int) ([]*models.CrawlJobConfig, error) {
	return nil, nil
}

type crawlTaskRepoForJobFake struct {
	bulkCreateFn func(ctx context.Context, entities []models.CrawlTask) ([]valueobjects.CrawlTaskID, error)
}

func (f crawlTaskRepoForJobFake) Create(context.Context, models.CrawlTask) (valueobjects.CrawlTaskID, error) {
	return valueobjects.CrawlTaskID{}, nil
}
func (f crawlTaskRepoForJobFake) BulkCreate(ctx context.Context, entities []models.CrawlTask) ([]valueobjects.CrawlTaskID, error) {
	return f.bulkCreateFn(ctx, entities)
}
func (f crawlTaskRepoForJobFake) Get(context.Context, valueobjects.CrawlTaskID) (*models.CrawlTask, error) {
	return nil, nil
}
func (f crawlTaskRepoForJobFake) Update(context.Context, models.CrawlTask) error { return nil }
func (f crawlTaskRepoForJobFake) ListByJob(context.Context, valueobjects.CrawlJobID) ([]*models.CrawlTask, error) {
	return nil, nil
}
func (f crawlTaskRepoForJobFake) ListByStatus(context.Context, models.TaskStatus, int) ([]*models.CrawlTask, error) {
	return nil, nil
}
func (f crawlTaskRepoForJobFake) ListWithCursor(context.Context, service.ListTasksByJobQuery) (*service.ListTasksResult, error) {
	return nil, nil
}
func (f crawlTaskRepoForJobFake) GetAnalyticsByJob(context.Context, valueobjects.CrawlJobID) (*service.TaskAnalytics, error) {
	return nil, nil
}
func (f crawlTaskRepoForJobFake) SetTaskResult(context.Context, valueobjects.CrawlTaskID, string, string, int64) error {
	return nil
}
func (f crawlTaskRepoForJobFake) ExistsByJobIDAndURL(context.Context, valueobjects.CrawlJobID, string) (bool, error) {
	return false, nil
}

type outboxRepoFake struct {
	createFn func(ctx context.Context, event models.OutboxEvent) error
}

func (f outboxRepoFake) Create(ctx context.Context, event models.OutboxEvent) error {
	return f.createFn(ctx, event)
}
func (f outboxRepoFake) BulkCreate(context.Context, []models.OutboxEvent) error { return nil }
func (f outboxRepoFake) FetchUnprocessedEvents(context.Context, int) ([]*models.OutboxEvent, error) {
	return nil, nil
}
func (f outboxRepoFake) MarkAsProcessed(context.Context, valueobjects.OutboxEventID) error { return nil }

type txManagerFake struct {
	runFn func(ctx context.Context, exec persistence.Handler) error
}

func (f txManagerFake) ReadCommitted(ctx context.Context, exec persistence.Handler) error {
	return f.runFn(ctx, exec)
}

var _ crawljobrepo.CrawlJobRepository = crawlJobRepoFake{}
var _ crawljobconfigrepo.CrawlJobConfigRepository = crawlJobConfigRepoFake{}
var _ crawltaskrepo.CrawlTaskRepository = crawlTaskRepoForJobFake{}
var _ outboxrepo.OutboxRepository = outboxRepoFake{}

func TestCreateCrawlJob_ValidatesSeedsAndAllowedPatterns(t *testing.T) {
	t.Parallel()

	svc := NewService(
		crawlJobRepoFake{
			createFn: func(ctx context.Context, entity models.CrawlJob) (valueobjects.CrawlJobID, error) {
				return valueobjects.CrawlJobID{}, nil
			},
			getFn: func(ctx context.Context, id valueobjects.CrawlJobID) (*models.CrawlJob, error) { return nil, nil },
			listFn: func(ctx context.Context, query service.ListCrawlJobsQuery) (*service.ListCrawlJobsResult, error) {
				return nil, nil
			},
		},
		crawlJobConfigRepoFake{
			createFn: func(ctx context.Context, entity models.CrawlJobConfig) (valueobjects.ID, error) {
				return valueobjects.GenerateID(), nil
			},
			getFn: func(ctx context.Context, id valueobjects.ID) (*models.CrawlJobConfig, error) { return nil, nil },
		},
		crawlTaskRepoForJobFake{
			bulkCreateFn: func(ctx context.Context, entities []models.CrawlTask) ([]valueobjects.CrawlTaskID, error) {
				return nil, nil
			},
		},
		outboxRepoFake{createFn: func(ctx context.Context, event models.OutboxEvent) error { return nil }},
		txManagerFake{runFn: func(ctx context.Context, exec persistence.Handler) error { return exec(ctx) }},
		nil,
	)

	_, err := svc.CreateCrawlJob(context.Background(), service.CreateCrawlJobCommand{
		Config: models.CrawlJobConfig{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "seeds list cannot be empty")
}

func TestCreateCrawlJob_CreatesConfigJobTasksAndOutbox(t *testing.T) {
	t.Parallel()

	var createdTasks []models.CrawlTask
	var outboxEvents []models.OutboxEvent
	var trimmedPatterns []string
	var createdJobID valueobjects.CrawlJobID
	configID := valueobjects.GenerateID()
	createdJobID = valueobjects.GenerateCrawlJobID()

	svc := NewService(
		crawlJobRepoFake{
			createFn: func(ctx context.Context, entity models.CrawlJob) (valueobjects.CrawlJobID, error) {
				assert.Equal(t, configID, entity.JobConfigID)
				assert.Equal(t, models.TaskStatusInProgress, entity.Status)
				assert.Equal(t, models.ExportStatusNotStarted, entity.ExportStatus)
				return createdJobID, nil
			},
			getFn: func(ctx context.Context, id valueobjects.CrawlJobID) (*models.CrawlJob, error) { return nil, nil },
			listFn: func(ctx context.Context, query service.ListCrawlJobsQuery) (*service.ListCrawlJobsResult, error) {
				return nil, nil
			},
		},
		crawlJobConfigRepoFake{
			createFn: func(ctx context.Context, entity models.CrawlJobConfig) (valueobjects.ID, error) {
				trimmedPatterns = entity.Scopes.AllowedURLPatterns
				assert.NotEmpty(t, entity.ID.String())
				return configID, nil
			},
			getFn: func(ctx context.Context, id valueobjects.ID) (*models.CrawlJobConfig, error) { return nil, nil },
		},
		crawlTaskRepoForJobFake{
			bulkCreateFn: func(ctx context.Context, entities []models.CrawlTask) ([]valueobjects.CrawlTaskID, error) {
				createdTasks = append(createdTasks, entities...)
				ids := make([]valueobjects.CrawlTaskID, 0, len(entities))
				for _, task := range entities {
					ids = append(ids, task.ID)
				}
				return ids, nil
			},
		},
		outboxRepoFake{
			createFn: func(ctx context.Context, event models.OutboxEvent) error {
				outboxEvents = append(outboxEvents, event)
				return nil
			},
		},
		txManagerFake{runFn: func(ctx context.Context, exec persistence.Handler) error { return exec(ctx) }},
		nil,
	)

	jobID, err := svc.CreateCrawlJob(context.Background(), service.CreateCrawlJobCommand{
		Config: models.CrawlJobConfig{
			Name:  "job",
			Seeds: []models.Seed{{Url: "https://example.com/a"}, {Url: "https://example.com/b"}},
			Scopes: models.ScopeRules{
				AllowedURLPatterns: []string{"  https://example.com/*  ", ""},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, createdJobID, jobID)
	assert.Equal(t, []string{"https://example.com/*"}, trimmedPatterns)
	require.Len(t, createdTasks, 2)
	require.Len(t, outboxEvents, 2)
	for i, task := range createdTasks {
		assert.Equal(t, createdJobID, task.JobID)
		assert.Equal(t, models.TaskStatusInProgress, task.Status)
		assert.Equal(t, uint64(0), task.Depth)

		var payload events.TaskEnqueuedEvent
		require.NoError(t, json.Unmarshal(outboxEvents[i].Payload, &payload))
		assert.Equal(t, task.ID.String(), payload.TaskID)
		assert.Equal(t, task.URL, payload.URL)
	}
}

func TestCreateCrawlJob_WrapsRepositoryErrors(t *testing.T) {
	t.Parallel()

	svc := NewService(
		crawlJobRepoFake{
			createFn: func(ctx context.Context, entity models.CrawlJob) (valueobjects.CrawlJobID, error) {
				return valueobjects.CrawlJobID{}, errors.New("job create failed")
			},
			getFn: func(ctx context.Context, id valueobjects.CrawlJobID) (*models.CrawlJob, error) { return nil, nil },
			listFn: func(ctx context.Context, query service.ListCrawlJobsQuery) (*service.ListCrawlJobsResult, error) { return nil, nil },
		},
		crawlJobConfigRepoFake{
			createFn: func(ctx context.Context, entity models.CrawlJobConfig) (valueobjects.ID, error) {
				return valueobjects.GenerateID(), nil
			},
			getFn: func(ctx context.Context, id valueobjects.ID) (*models.CrawlJobConfig, error) { return nil, nil },
		},
		crawlTaskRepoForJobFake{bulkCreateFn: func(ctx context.Context, entities []models.CrawlTask) ([]valueobjects.CrawlTaskID, error) { return nil, nil }},
		outboxRepoFake{createFn: func(ctx context.Context, event models.OutboxEvent) error { return nil }},
		txManagerFake{runFn: func(ctx context.Context, exec persistence.Handler) error { return exec(ctx) }},
		nil,
	)

	_, err := svc.CreateCrawlJob(context.Background(), service.CreateCrawlJobCommand{
		Config: models.CrawlJobConfig{Seeds: []models.Seed{{Url: "https://example.com"}}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create crawl job")
}

func TestGetAndListCrawlJobs_ValidateAndWrapErrors(t *testing.T) {
	t.Parallel()

	jobID := valueobjects.GenerateCrawlJobID()
	createdAt := time.Now().UTC()
	svc := NewService(
		crawlJobRepoFake{
			createFn: func(ctx context.Context, entity models.CrawlJob) (valueobjects.CrawlJobID, error) { return valueobjects.CrawlJobID{}, nil },
			getFn: func(ctx context.Context, id valueobjects.CrawlJobID) (*models.CrawlJob, error) {
				assert.Equal(t, jobID, id)
				return &models.CrawlJob{ID: id, JobConfigID: valueobjects.GenerateID(), CreatedAt: createdAt}, nil
			},
			listFn: func(ctx context.Context, query service.ListCrawlJobsQuery) (*service.ListCrawlJobsResult, error) {
				assert.Equal(t, 100, query.Limit)
				return nil, errors.New("list fail")
			},
		},
		crawlJobConfigRepoFake{},
		crawlTaskRepoForJobFake{},
		outboxRepoFake{},
		txManagerFake{runFn: func(ctx context.Context, exec persistence.Handler) error { return exec(ctx) }},
		nil,
	)

	job, err := svc.GetCrawlJob(context.Background(), service.GetCrawlJobQuery{ID: jobID.String()})
	require.NoError(t, err)
	assert.Equal(t, jobID, job.ID)

	job, err = svc.GetCrawlJob(context.Background(), service.GetCrawlJobQuery{ID: "bad"})
	require.Error(t, err)
	assert.Nil(t, job)

	status := "INVALID"
	_, err = svc.ListCrawlJobs(context.Background(), service.ListCrawlJobsQuery{
		Filter: service.ListCrawlJobsFilter{Status: &status},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")

	from := createdAt
	to := createdAt.Add(-time.Hour)
	_, err = svc.ListCrawlJobs(context.Background(), service.ListCrawlJobsQuery{
		Filter: service.ListCrawlJobsFilter{CreatedFrom: &from, CreatedTo: &to},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "created_from")

	_, err = svc.ListCrawlJobs(context.Background(), service.ListCrawlJobsQuery{Limit: 999})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list crawl jobs")
}
