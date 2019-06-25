package gpool

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// default config
const (
	DefaultCapacity    = 1000
	DefaultIdleTImeout = 60 * time.Second
)

// Config 配置
type Config struct {
	Capacity    int
	IdleTimeout time.Duration
}

// Pool the goroutine pool
type Pool struct {
	cfg    *Config
	once   sync.Once
	ctx    context.Context
	cancel context.CancelFunc

	running int32 // goroutines running count
	idle    int32 // goroutines has create but in idle
	work    chan func(context.Context)
}

// New a pool
func New(ctx context.Context, c ...*Config) *Pool {
	ctx, cancel := context.WithCancel(ctx)
	if len(c) == 0 {
		c = append(c, &Config{Capacity: DefaultCapacity, IdleTimeout: DefaultIdleTImeout})
	}

	cfg := c[0]
	return &Pool{
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
		work:   make(chan func(context.Context), cfg.Capacity),
	}
}

// Submit submits a task
func (this *Pool) Submit(f func(context.Context)) error {
	defer func() {
		if r := recover(); r != nil {
			// log
		}
	}()

	this.work <- f
	if atomic.LoadInt32(&this.idle) == 0 {
		go this.run()
	}

	return nil
}

// Len returns the currently running goroutines
func (this *Pool) Len() int {
	return int(atomic.LoadInt32(&this.running))
}

// Idle return the goroutines has create but in idle
func (this *Pool) Idle() int {
	return int(atomic.LoadInt32(&this.idle))
}

// Free return the available goroutines can create
func (this *Pool) Free() int {
	return this.cfg.Capacity - int(atomic.LoadInt32(&this.running))
}

// Cap tha capacity of goroutines the pool can create
func (this *Pool) Cap() int {
	return this.cfg.Capacity
}

// Close the pool
func (this *Pool) Close() {
	this.once.Do(func() {
		close(this.work)
		this.cancel()
	})
}

func (this *Pool) run() {
	t := time.NewTimer(this.cfg.IdleTimeout)
	defer func() {
		atomic.AddInt32(&this.running, -1)
		t.Stop()
		if r := recover(); r != nil {
			// log
		}
	}()
	atomic.AddInt32(&this.running, 1)
	atomic.AddInt32(&this.idle, 1)
	for {
		select {
		case <-t.C:
			atomic.AddInt32(&this.idle, -1)
			return
		case f, ok := <-this.work:
			atomic.AddInt32(&this.idle, -1)
			if !ok {
				return
			}
			f(this.ctx)
			t.Reset(this.cfg.IdleTimeout)
			atomic.AddInt32(&this.idle, 1)
		}
	}
}
