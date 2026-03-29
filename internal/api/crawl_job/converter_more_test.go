package crawljob

import (
	"testing"
	"time"

	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	crawlergrpc "distributed-crawler/pkg/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtoConversion_RoundTripForJobConfig(t *testing.T) {
	t.Parallel()

	index := 1
	config := &models.CrawlJobConfig{
		ID:   valueobjects.GenerateID(),
		Name: "job",
		ExtractionSpec: models.ExtractionSpec{
			Fields: []models.FieldSpec{{
				Name:     "title",
				Type:     models.ValueString,
				Required: true,
				Extractor: models.ExtractorSpec{
					Selector:  "h1",
					Attribute: "text",
					Multiple:  true,
					Index:     &index,
				},
				Transforms: []models.TransformSpec{{Op: models.OpTrim, Arg: "x"}},
			}},
			Items: &models.ItemsSpec{
				ContainerSelector: ".item",
				Fields: []models.FieldSpec{{
					Name:      "url",
					Type:      models.ValueURL,
					Extractor: models.ExtractorSpec{Selector: "a", Attribute: "href"},
				}},
			},
			Pagination: []models.PaginationSpec{{Name: "next", Selector: ".next", Attribute: "href", Multiple: true}},
		},
		Scopes: models.ScopeRules{
			MaxDepth:           2,
			AllowedDomains:     []string{"example.com"},
			DenyUrlPatterns:    []string{"/logout"},
			AllowedURLPatterns: []string{"https://example.com/*"},
		},
		Seeds:            []models.Seed{{Url: "https://example.com"}},
		RateLimit:        models.RateLimitPolicy{Rps: 1.5},
		Retries:          models.RetryPolicy{MaxAttempts: 3, BackoffInitialMs: 100, BackoffMultiplier: 2},
		Auth:             models.AuthOptions{Cookie: "a=b", BasicUser: "u", BasicPassword: "p", BearerToken: "t"},
		Schedule:         models.ScheduleOptions{Cron: "* * * * *"},
		JobType:          models.JobTypeScheduled,
		RespectRobotsTxt: true,
		CrawlMode:        models.CrawlModeLinksOnly,
	}

	protoCfg := ToProtoCrawlJobConfig(config)
	require.NotNil(t, protoCfg)
	assert.Equal(t, crawlergrpc.JobType_JOB_TYPE_SCHEDULED, protoCfg.JobType)
	assert.Equal(t, crawlergrpc.CrawlMode_CRAWL_MODE_LINKS_ONLY, protoCfg.CrawlMode)

	roundTrip := FromProtoCrawlJobConfig(protoCfg)
	assert.Equal(t, config.Name, roundTrip.Name)
	assert.Equal(t, config.Scopes.AllowedURLPatterns, roundTrip.Scopes.AllowedURLPatterns)
	assert.Equal(t, config.Auth.Cookie, roundTrip.Auth.Cookie)
	assert.Equal(t, config.Schedule.Cron, roundTrip.Schedule.Cron)
	assert.Equal(t, config.JobType, roundTrip.JobType)
	assert.Equal(t, config.CrawlMode, roundTrip.CrawlMode)
}

func TestProtoConversion_DefaultsAndNilHandling(t *testing.T) {
	t.Parallel()

	assert.Nil(t, ToProtoItemsSpec(nil))
	assert.Nil(t, ToProtoCrawlJob(nil))
	assert.Nil(t, ToProtoCrawlTask(nil))
	assert.Nil(t, ToProtoCrawlJobConfig(nil))
	assert.Equal(t, models.CrawlMode(""), FromProtoCrawlMode(crawlergrpc.CrawlMode_CRAWL_MODE_UNSPECIFIED))
	assert.Equal(t, models.JobTypeOnce, FromProtoJobType(crawlergrpc.JobType_JOB_TYPE_UNSPECIFIED))
	assert.Equal(t, crawlergrpc.CrawlMode_CRAWL_MODE_UNSPECIFIED, ToProtoCrawlMode(models.CrawlMode("bad")))
}

func TestToProtoCrawlJobAndTask_OptionalFields(t *testing.T) {
	t.Parallel()

	jobID := valueobjects.GenerateCrawlJobID()
	taskID := valueobjects.GenerateCrawlTaskID()
	cfgID := valueobjects.GenerateID()
	name := "scheduled"
	completedAt := time.Now().UTC().Round(0)
	exportJSON := "export.json"
	exportCSV := "export.csv"
	finalURL := "https://example.com/final"
	resultKey := "results/task.json"
	errorMessage := "boom"

	protoJob := ToProtoCrawlJob(&models.CrawlJob{
		ID:            jobID,
		JobConfigID:   cfgID,
		Name:          &name,
		Status:        models.TaskStatusCompleted,
		CreatedAt:     completedAt,
		CompletedAt:   &completedAt,
		ExportJSONKey: &exportJSON,
		ExportCSVKey:  &exportCSV,
		ExportedAt:    &completedAt,
		ExportStatus:  models.ExportStatusCompleted,
	})
	require.NotNil(t, protoJob)
	assert.Equal(t, &exportJSON, protoJob.ExportJsonKey)
	assert.Equal(t, &exportCSV, protoJob.ExportCsvKey)

	protoTask := ToProtoCrawlTask(&models.CrawlTask{
		ID:             taskID,
		JobID:          jobID,
		URL:            "https://example.com",
		Status:         models.TaskStatusFailed,
		EnqueuedAt:     completedAt,
		Depth:          1,
		MinioObjectKey: "pages/task.html",
		FinalURL:       &finalURL,
		ResultObjectKey: &resultKey,
		ErrorMessage:   &errorMessage,
	})
	require.NotNil(t, protoTask)
	assert.Equal(t, &finalURL, protoTask.FinalUrl)
	assert.Equal(t, &resultKey, protoTask.ResultObjectKey)
	assert.Equal(t, &errorMessage, protoTask.ErrorMessage)
}

