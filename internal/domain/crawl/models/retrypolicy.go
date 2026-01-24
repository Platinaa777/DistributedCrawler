package models

type RetryPolicy struct {
	MaxAttempts       uint64
	BackoffInitialMs  uint64
	BackoffMultiplier float64
}
