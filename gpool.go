package gpool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// the default config parameter
const (
	DefaultCapacity      = 100000
	DefaultKeepAliveTime = 1 * time.Second
	miniCleanupTime      = 100 * time.Millisecond
)

// ErrClosed indicate the pool has closed
var ErrClosed = errors.New("pool has closed")

// ErrInvalidFunc indicate the task function is invalid
var ErrInvalidFunc = errors.New("invalid function, must not be nil")

// TaskFunc task function define
type TaskFunc func(arg interface{})

// Config the pool config parameter
type Config struct {
	Capacity    int
	IdleTimeout time.Duration
}

// Pool the goroutine pool
type Pool struct {
	cfg    *Config
	ctx    context.Context
	cancel context.CancelFunc

	running int32 // goroutines running count

	closeDone uint32

	mux   sync.Mutex
	cond  *sync.Cond
	idle  *list
	cache *sync.Pool

	panicFunc func()
}

// New a pool with the config if there is ,other use default config
func New(c ...*Config) *Pool {
	if len(c) == 0 {
		c = append(c, &Config{Capacity: DefaultCapacity, IdleTimeout: DefaultKeepAliveTime})
	} else if c[0].Capacity < 0 {
		c[0].Capacity = DefaultCapacity
	}

	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		cfg:    c[0],
		ctx:    ctx,
		cancel: cancel,

		idle: newList(),
	}
	p.cond = sync.NewCond(&p.mux)
	p.cache = &sync.Pool{
		New: func() interface{} { return &work{itm: make(chan item, 1), pool: p} },
	}
	go p.cleanUp()
	return p
}

func (this *Pool) cleanUp() {
	tick := time.NewTimer(this.cfg.IdleTimeout)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			nearTimeout := this.cfg.IdleTimeout
			now := time.Now()
			this.mux.Lock()
			var next *work
			for e := this.idle.Front(); e != nil; e = next {
				if nearTimeout = now.Sub(e.markTime); nearTimeout < this.cfg.IdleTimeout {
					break
				}
				next = e.Next() // save before delete
				this.idle.remove(e).itm <- item{}
			}
			this.mux.Unlock()
			if nearTimeout < miniCleanupTime {
				nearTimeout = miniCleanupTime
			}
			fmt.Println(nearTimeout)
			tick.Reset(nearTimeout)
		case <-this.ctx.Done():
			this.mux.Lock()
			for e := this.idle.Front(); e != nil; e = e.Next() {
				e.itm <- item{}
			}
			this.idle = nil
			this.mux.Unlock()
			return
		}
	}
}

func (this *Pool) SetPanicHandler(f func()) {
	this.panicFunc = f
}

// Len returns the currently running goroutines
func (this *Pool) Len() int {
	return int(atomic.LoadInt32(&this.running))
}

// Idle return the goroutines has running but in idle(no task work)
func (this *Pool) Idle() int {
	var cnt int
	this.mux.Lock()
	if this.idle != nil {
		cnt = this.idle.Len()
	}
	this.mux.Unlock()
	return cnt
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
func (this *Pool) Close() error {
	if atomic.LoadUint32(&this.closeDone) == 1 {
		return nil
	}

	this.mux.Lock()
	if this.closeDone == 0 { // check again,make sure
		this.cancel()
		atomic.StoreUint32(&this.closeDone, 1)
	}
	this.mux.Unlock()
	return nil
}

// Submit submits a task with arg
func (this *Pool) Submit(f TaskFunc, arg interface{}) error {
	var w *work

	if f == nil {
		return ErrInvalidFunc
	}

	if atomic.LoadUint32(&this.closeDone) == 1 {
		return ErrClosed
	}

	this.mux.Lock()
	if this.closeDone == 1 || this.idle == nil { // check again,make sure
		this.mux.Unlock()
		return ErrClosed
	}

	itm := item{f, arg}
	if w = this.idle.Front(); w != nil {
		this.idle.Remove(w)
		this.mux.Unlock()
		w.itm <- itm
		return nil
	}

	if this.Free() > 0 {
		this.mux.Unlock()
		w = this.cache.Get().(*work)
		w.run(itm)
		return nil
	}

	for {
		this.cond.Wait()
		if w = this.idle.Front(); w != nil {
			this.idle.Remove(w)
			break
		}
	}
	this.mux.Unlock()
	w.itm <- itm
	return nil
}

// push the running goroutine to idle pool
func (this *Pool) push(w *work) error {
	if atomic.LoadUint32(&this.closeDone) == 1 { // quick check
		return ErrClosed
	}

	w.markTime = time.Now()
	this.mux.Lock()
	if this.closeDone == 1 { // check again,make sure
		this.mux.Unlock()
		return ErrClosed
	}
	this.idle.PushBack(w)
	this.cond.Signal()
	this.mux.Unlock()
	return nil
}

func (this *work) run(f item) {
	atomic.AddInt32(&this.pool.running, 1)
	go func() {
		defer func() {
			atomic.AddInt32(&this.pool.running, -1)
			this.pool.cache.Put(this)
			if r := recover(); r != nil && this.pool.panicFunc != nil {
				this.pool.panicFunc()
			}
		}()

		for {
			f.task(f.arg)
			if this.pool.push(this) != nil {
				return
			}
			if f = <-this.itm; f.task == nil {
				return
			}
		}
	}()
}
