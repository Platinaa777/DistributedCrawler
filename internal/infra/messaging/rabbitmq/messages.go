package rabbitmq

import "time"

// CrawlTaskMessage represents a crawl task message sent to RabbitMQ
type CrawlTaskMessage struct {
	TaskID     string    `json:"task_id"`
	JobID      string    `json:"job_id"`
	URL        string    `json:"url"`
	EnqueuedAt time.Time `json:"enqueued_at"`
}
