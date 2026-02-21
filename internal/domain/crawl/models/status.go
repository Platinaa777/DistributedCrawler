package models

type TaskStatus string

const (
	TaskStatusInProgress TaskStatus = "InProgress"
	TaskStatusFetched    TaskStatus = "Fetched"   // Page fetched and stored in MinIO
	TaskStatusParsed     TaskStatus = "Parsed"    // Page parsed and data extracted
	TaskStatusCompleted  TaskStatus = "Completed"
	TaskStatusFailed     TaskStatus = "Failed"
	TaskStatusSkipped    TaskStatus = "Skipped" // Skipped due to deduplication
)

func (s TaskStatus) String() string {
	return string(s)
}

func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusInProgress, TaskStatusFetched, TaskStatusCompleted, TaskStatusFailed, TaskStatusParsed, TaskStatusSkipped:
		return true
	}
	return false
}

func AllTaskStatuses() []TaskStatus {
	return []TaskStatus{
		TaskStatusInProgress,
		TaskStatusFetched,
		TaskStatusParsed,
		TaskStatusCompleted,
		TaskStatusFailed,
		TaskStatusSkipped,
	}
}

func AllTaskStatusesString() string {
	statuses := AllTaskStatuses()
	result := ""
	for i, status := range statuses {
		if i > 0 {
			result += ", "
		}
		result += status.String()
	}
	return result
}

// ExportStatus represents the status of job export (Part B - ExportWorker)
type ExportStatus string

const (
	ExportStatusNotStarted ExportStatus = "NOT_STARTED"
	ExportStatusInProgress ExportStatus = "IN_PROGRESS"
	ExportStatusCompleted  ExportStatus = "COMPLETED"
	ExportStatusFailed     ExportStatus = "FAILED"
)

func (s ExportStatus) String() string {
	return string(s)
}

func (s ExportStatus) IsValid() bool {
	switch s {
	case ExportStatusNotStarted, ExportStatusInProgress, ExportStatusCompleted, ExportStatusFailed:
		return true
	}
	return false
}
