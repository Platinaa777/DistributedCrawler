package events

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBaseEventAndTaskEnqueuedEvent(t *testing.T) {
	t.Parallel()

	base := NewBaseEvent(EventTypeTaskEnqueued)
	require.NotEmpty(t, base.ID)
	assert.Equal(t, EventTypeTaskEnqueued, base.Type)
	assert.False(t, base.OccurredAt.IsZero())

	enqueuedAt := time.Now().UTC().Round(0)
	event := NewTaskEnqueuedEvent("task-1", "job-1", "https://example.com", enqueuedAt)
	assert.Equal(t, EventTypeTaskEnqueued, event.Type)
	assert.Equal(t, "task-1", event.TaskID)
	assert.Equal(t, "job-1", event.JobID)
	assert.Equal(t, "https://example.com", event.URL)
	assert.True(t, event.EnqueuedAt.Equal(enqueuedAt))
}

