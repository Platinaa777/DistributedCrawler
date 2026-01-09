package rabbitmq

import "time"

// CrawlTaskMessage represents a crawl task message sent to crawl_queue
// Consumed by FetchWorker
type CrawlTaskMessage struct {
	TaskID     string    `json:"task_id"`
	JobID      string    `json:"job_id"`
	URL        string    `json:"url"`
	EnqueuedAt time.Time `json:"enqueued_at"`
}

// ParsingTaskMessage represents a parsing task message sent to parsing_queue
// Published by FetchWorker after successful fetch, consumed by ParserWorker
type ParsingTaskMessage struct {
	TaskID     string    `json:"task_id"`
	JobID      string    `json:"job_id"`
	EnqueuedAt time.Time `json:"enqueued_at"`
}
