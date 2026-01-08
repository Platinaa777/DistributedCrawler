package models

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "Pending"
	TaskStatusRunning   TaskStatus = "Running"
	TaskStatusCompleted TaskStatus = "Completed"
	TaskStatusFailed    TaskStatus = "Failed"
)

func (s TaskStatus) String() string {
	return string(s)
}

func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusPending, TaskStatusRunning, TaskStatusCompleted, TaskStatusFailed:
		return true
	}
	return false
}
