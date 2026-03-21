package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskAndExportStatusHelpers(t *testing.T) {
	t.Parallel()

	assert.True(t, TaskStatusParsed.IsValid())
	assert.False(t, TaskStatus("BAD").IsValid())
	assert.Equal(t, "Parsed", TaskStatusParsed.String())
	assert.Len(t, AllTaskStatuses(), 6)
	assert.Contains(t, AllTaskStatusesString(), "Skipped")

	assert.True(t, ExportStatusCompleted.IsValid())
	assert.False(t, ExportStatus("BAD").IsValid())
	assert.Equal(t, "COMPLETED", ExportStatusCompleted.String())
}

func TestCrawlJobMarkAsExported(t *testing.T) {
	t.Parallel()

	job := &CrawlJob{}
	exportedAt := time.Now().UTC().Round(0)
	job.MarkAsExported("export.json", "export.csv", exportedAt)

	require.NotNil(t, job.ExportJSONKey)
	require.NotNil(t, job.ExportCSVKey)
	require.NotNil(t, job.ExportedAt)
	require.NotNil(t, job.CompletedAt)
	assert.Equal(t, "export.json", *job.ExportJSONKey)
	assert.Equal(t, "export.csv", *job.ExportCSVKey)
	assert.Equal(t, ExportStatusCompleted, job.ExportStatus)
	assert.Equal(t, TaskStatusCompleted, job.Status)
	assert.True(t, job.CompletedAt.Equal(exportedAt))
}

func TestScopeRulesCompileAllowedURLPatterns(t *testing.T) {
	t.Parallel()

	re, err := CompileAllowedURLPattern(" https://example.com/* ")
	require.NoError(t, err)
	require.NotNil(t, re)
	assert.True(t, re.MatchString("https://example.com/a/b"))
	assert.False(t, re.MatchString("https://other.com/a"))

	re, err = CompileAllowedURLPattern(" ")
	require.NoError(t, err)
	assert.Nil(t, re)

	patterns, err := CompileAllowedURLPatterns([]string{"https://example.com/*", "", "https://example.org/*"})
	require.NoError(t, err)
	require.Len(t, patterns, 2)
}

