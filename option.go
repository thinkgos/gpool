package gpool

import (
	"time"
)

type Option func(pool *Pool)

//// Config the pool config parameter
//type Config struct {
//	Capacity        int
//	SurvivalTime    time.Duration
//	MiniCleanupTime time.Duration // mini cleanup time
//}

func WithCapacity(cap int32) Option {
	return func(pool *Pool) {
		pool.capacity = cap
	}
}

func WithSurvivalTime(t time.Duration) Option {
	return func(pool *Pool) {
		pool.survivalTime = t
	}
}

func WithMiniCleanupTime(t time.Duration) Option {
	return func(pool *Pool) {
		pool.miniCleanupTime = t
	}
}
