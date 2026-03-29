package events

import "time"

// TaskEnqueuedEvent is emitted when a crawl task is created and needs to be processed
type TaskEnqueuedEvent struct {
	BaseEvent
	TaskID       string            `json:"task_id"`
	JobID        string            `json:"job_id"`
	URL          string            `json:"url"`
	EnqueuedAt   time.Time         `json:"enqueued_at"`
	TraceContext map[string]string `json:"trace_context,omitempty"`
	// TargetQueue is the crawl queue this task should be published to.
	// Set at task creation time by the queue routing policy.
	// Empty means fall back to the outbox publisher's configured queue.
	TargetQueue string `json:"target_queue,omitempty"`
}

// NewTaskEnqueuedEvent creates a new TaskEnqueuedEvent
func NewTaskEnqueuedEvent(taskID, jobID, url string, enqueuedAt time.Time) TaskEnqueuedEvent {
	return TaskEnqueuedEvent{
		BaseEvent:  NewBaseEvent(EventTypeTaskEnqueued),
		TaskID:     taskID,
		JobID:      jobID,
		URL:        url,
		EnqueuedAt: enqueuedAt,
	}
}
