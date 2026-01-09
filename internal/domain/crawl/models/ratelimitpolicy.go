package models

type RateLimitPolicy struct {
	MaxConcurrency uint64
	JitterMs       uint64
	Rps            float64
}
