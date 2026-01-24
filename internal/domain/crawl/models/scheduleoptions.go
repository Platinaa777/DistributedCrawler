package models

import "time"

type ScheduleOptions struct {
	Cron      string
	LastRunAt *time.Time
	NextRunAt *time.Time
}
