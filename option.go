package gpool

import (
	"time"
)

// Option pool option
type Option func(pool *Pool)

// WithCapacity set goroutines capacity
func WithCapacity(cap int32) Option {
	return func(pool *Pool) {
		pool.capacity = cap
	}
}

// WithSurvivalTime set goroutines survival time
func WithSurvivalTime(t time.Duration) Option {
	return func(pool *Pool) {
		pool.survivalTime = t
	}
}
