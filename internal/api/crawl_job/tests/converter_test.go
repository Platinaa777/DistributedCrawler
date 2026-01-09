package tests

import (
	crawljob "distributed-crawler/internal/api/crawl_job"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	crawlergrpc "distributed-crawler/pkg/v1"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestToProtoCrawlJob(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	completedTime := now.Add(1 * time.Hour)

	configID := valueobjects.GenerateID()
	jobName := gofakeit.AppName()
	emptyName := ""

	tests := []struct {
		name     string
		input    *models.CrawlJob
		expected *crawlergrpc.CrawlJob
	}{
		{
			name:     "nil input returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name: "valid job without completed time",
			input: &models.CrawlJob{
				ID:          valueobjects.GenerateCrawlJobID(),
				JobConfigID: configID,
				JobConfig: &models.CrawlJobConfig{
					ID:   configID,
					Name: jobName,
				},
				Status:      models.TaskStatus(gofakeit.RandomString([]string{"pending", "running", "completed", "failed"})),
				CreatedAt:   now,
				CompletedAt: nil,
			},
			expected: &crawlergrpc.CrawlJob{
				// ID, JobConfigID, and Status will be validated separately
				CreatedAt:   timestamppb.New(now),
				CompletedAt: nil,
			},
		},
		{
			name: "valid job with completed time",
			input: &models.CrawlJob{
				ID:          valueobjects.GenerateCrawlJobID(),
				JobConfigID: configID,
				JobConfig: &models.CrawlJobConfig{
					ID:   configID,
					Name: gofakeit.JobTitle(),
				},
				Status:      models.TaskStatusCompleted,
				CreatedAt:   now,
				CompletedAt: &completedTime,
			},
			expected: &crawlergrpc.CrawlJob{
				// ID and JobConfigID will be validated separately
				Status:      "completed",
				CreatedAt:   timestamppb.New(now),
				CompletedAt: timestamppb.New(completedTime),
			},
		},
		{
			name: "job with random generated data",
			input: &models.CrawlJob{
				ID:          valueobjects.GenerateCrawlJobID(),
				JobConfigID: configID,
				JobConfig: &models.CrawlJobConfig{
					ID:   configID,
					Name: gofakeit.Word(),
				},
				Status:      models.TaskStatus(gofakeit.RandomString([]string{"pending", "running"})),
				CreatedAt:   gofakeit.Date(),
				CompletedAt: nil,
			},
			expected: &crawlergrpc.CrawlJob{
				// Will be validated separately
			},
		},
		{
			name: "job with empty name in config",
			input: &models.CrawlJob{
				ID:          valueobjects.GenerateCrawlJobID(),
				JobConfigID: configID,
				JobConfig: &models.CrawlJobConfig{
					ID:   configID,
					Name: emptyName,
				},
				Status:      models.TaskStatusInProgress,
				CreatedAt:   now,
				CompletedAt: nil,
			},
			expected: &crawlergrpc.CrawlJob{
				Status:      "pending",
				CreatedAt:   timestamppb.New(now),
				CompletedAt: nil,
			},
		},
		{
			name: "job without config",
			input: &models.CrawlJob{
				ID:          valueobjects.GenerateCrawlJobID(),
				JobConfigID: configID,
				JobConfig:   nil,
				Status:      models.TaskStatusInProgress,
				CreatedAt:   now,
				CompletedAt: nil,
			},
			expected: &crawlergrpc.CrawlJob{
				Status:      "pending",
				CreatedAt:   timestamppb.New(now),
				CompletedAt: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := crawljob.ToProtoCrawlJob(tt.input)

			if tt.input == nil {
				require.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			assert.Equal(t, tt.input.ID.String(), result.Id)
			assert.Equal(t, tt.input.JobConfigID.String(), result.JobConfigId)
			assert.Equal(t, tt.input.Status.String(), result.Status)

			// Validate JobConfig
			if tt.input.JobConfig != nil {
				require.NotNil(t, result.JobConfig)
				assert.Equal(t, tt.input.JobConfig.Name, result.JobConfig.Name)
			} else {
				assert.Nil(t, result.JobConfig)
			}

			// Validate timestamps
			require.NotNil(t, result.CreatedAt)
			assert.True(t, result.CreatedAt.AsTime().Equal(tt.input.CreatedAt))

			// Validate CompletedAt
			if tt.input.CompletedAt == nil {
				assert.Nil(t, result.CompletedAt)
			} else {
				require.NotNil(t, result.CompletedAt)
				assert.True(t, result.CompletedAt.AsTime().Equal(*tt.input.CompletedAt))
			}
		})
	}
}

func TestToProtoCrawlJob_Properties(t *testing.T) {
	t.Parallel()

	t.Run("preserves UUID format", func(t *testing.T) {
		t.Parallel()

		jobID := valueobjects.GenerateCrawlJobID()
		configID := valueobjects.GenerateID()
		job := &models.CrawlJob{
			ID:          jobID,
			JobConfigID: configID,
			JobConfig: &models.CrawlJobConfig{
				ID:   configID,
				Name: gofakeit.BuzzWord(),
			},
			Status:    models.TaskStatusInProgress,
			CreatedAt: time.Now(),
		}

		result := crawljob.ToProtoCrawlJob(job)

		require.NotNil(t, result)
		parsedUUID, err := uuid.Parse(result.Id)
		require.NoError(t, err)
		assert.Equal(t, jobID.String(), parsedUUID.String())
	})

	t.Run("handles time precision correctly", func(t *testing.T) {
		t.Parallel()

		// Test with nanosecond precision
		preciseTime := time.Date(2024, 1, 15, 10, 30, 45, 123456789, time.UTC)
		configID := valueobjects.GenerateID()
		job := &models.CrawlJob{
			ID:          valueobjects.GenerateCrawlJobID(),
			JobConfigID: configID,
			JobConfig: &models.CrawlJobConfig{
				ID:   configID,
				Name: gofakeit.Company(),
			},
			Status:    models.TaskStatusCompleted,
			CreatedAt: preciseTime,
		}

		result := crawljob.ToProtoCrawlJob(job)

		require.NotNil(t, result)
		// timestamppb preserves nanosecond precision
		assert.Equal(t, preciseTime.UnixNano(), result.CreatedAt.AsTime().UnixNano())
	})

	t.Run("handles special characters in name", func(t *testing.T) {
		t.Parallel()

		specialNames := []string{
			"Job with spaces",
			"Job-with-dashes",
			"Job_with_underscores",
			"Job/with/slashes",
			"Job🚀with✨emojis",
			"Работа с кириллицей",
		}

		for _, name := range specialNames {
			configID := valueobjects.GenerateID()
			job := &models.CrawlJob{
				ID:          valueobjects.GenerateCrawlJobID(),
				JobConfigID: configID,
				JobConfig: &models.CrawlJobConfig{
					ID:   configID,
					Name: name,
				},
				Status:    models.TaskStatusInProgress,
				CreatedAt: time.Now(),
			}

			result := crawljob.ToProtoCrawlJob(job)

			require.NotNil(t, result)
			require.NotNil(t, result.JobConfig)
			assert.Equal(t, name, result.JobConfig.Name)
		}
	})
}

func BenchmarkToProtoCrawlJob(b *testing.B) {
	configID := valueobjects.GenerateID()
	job := &models.CrawlJob{
		ID:          valueobjects.GenerateCrawlJobID(),
		JobConfigID: configID,
		JobConfig: &models.CrawlJobConfig{
			ID:   configID,
			Name: gofakeit.AppName(),
		},
		Status:    models.TaskStatusInProgress,
		CreatedAt: time.Now(),
	}

	b.ResetTimer()
	for b.Loop() {
		_ = crawljob.ToProtoCrawlJob(job)
	}
}

func BenchmarkToProtoCrawlJob_WithCompletedAt(b *testing.B) {
	completedAt := time.Now()
	configID := valueobjects.GenerateID()
	job := &models.CrawlJob{
		ID:          valueobjects.GenerateCrawlJobID(),
		JobConfigID: configID,
		JobConfig: &models.CrawlJobConfig{
			ID:   configID,
			Name: gofakeit.AppName(),
		},
		Status:      models.TaskStatusCompleted,
		CreatedAt:   time.Now(),
		CompletedAt: &completedAt,
	}

	b.ResetTimer()
	for b.Loop() {
		_ = crawljob.ToProtoCrawlJob(job)
	}
}
