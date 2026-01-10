package models

type TaskStatus string

const (
	TaskStatusInProgress TaskStatus = "InProgress"
	TaskStatusParsed     TaskStatus = "Parsed"
	TaskStatusCompleted  TaskStatus = "Completed"
	TaskStatusFailed     TaskStatus = "Failed"
)

func (s TaskStatus) String() string {
	return string(s)
}

func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusInProgress, TaskStatusCompleted, TaskStatusFailed, TaskStatusParsed:
		return true
	}
	return false
}

func AllTaskStatuses() []TaskStatus {
	return []TaskStatus{
		TaskStatusInProgress,
		TaskStatusParsed,
		TaskStatusCompleted,
		TaskStatusFailed,
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
