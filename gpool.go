package gpool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// default config
const (
	DefaultCapacity    = 1024
	DefaultIdleTImeout = 1 * time.Second
)

var ErrClosed = errors.New("pool has closed")

// TaskFunc task function define
type TaskFunc func(interface{})

// Config the pool config
type Config struct {
	Capacity    int
	IdleTimeout time.Duration
}

// Pool the goroutine pool
type Pool struct {
	cfg    *Config
	ctx    context.Context
	cancel context.CancelFunc

	closeDone uint32

	running int32 // goroutines running count
	mux     sync.Mutex
	cond    *sync.Cond
	idle    *list
	cache   *sync.Pool
}

// New a pool with the config
func New(c ...*Config) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	if len(c) == 0 {
		c = append(c, &Config{Capacity: DefaultCapacity, IdleTimeout: DefaultIdleTImeout})
	}

	cfg := c[0]
	p := &Pool{
		cfg:    cfg,
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
			now := time.Now()
			this.mux.Lock()
			var next *work
			for e := this.idle.Front(); e != nil; e = next {
				if now.Sub(e.markTime) < this.cfg.IdleTimeout {
					break
				}
				next = e.Next() // save before delete
				this.idle.remove(e).itm <- item{}
			}
			this.mux.Unlock()
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

// Submit submits a task
func (this *Pool) Submit(f TaskFunc) error {
	return this.submit(f, nil)
}

// Submit submits a task
func (this *Pool) Submit2(f TaskFunc, arg interface{}) error {
	return this.submit(f, arg)
}

// Len returns the currently running goroutines
func (this *Pool) Len() int {
	return int(atomic.LoadInt32(&this.running))
}

// Idle return the goroutines has create but in idle
func (this *Pool) Idle() int {
	this.mux.Lock()
	cnt := this.idle.Len()
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

func (this *Pool) submit(f TaskFunc, arg interface{}) error {
	var w *work

	if atomic.LoadUint32(&this.closeDone) == 1 {
		return ErrClosed
	}

	this.mux.Lock()
	if this.closeDone == 1 || this.idle == nil {
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

	if int(atomic.LoadInt32(&this.running)) < this.Cap() {
		this.mux.Unlock()
		w = this.cache.Get().(*work)
		go w.run(itm)
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

func (this *Pool) push(w *work) error {
	// quick check
	if atomic.LoadUint32(&this.closeDone) == 1 {
		return ErrClosed
	}

	w.markTime = time.Now()
	this.mux.Lock()
	if this.closeDone == 1 {
		this.mux.Unlock()
		return ErrClosed
	}
	this.idle.PushBack(w)
	this.cond.Signal()
	this.mux.Unlock()
	return nil
}

func (this *work) run(f item) {
	defer func() {
		atomic.AddInt32(&this.pool.running, -1)
		this.pool.cache.Put(this)
		if r := recover(); r != nil {
			// log
		}
	}()

	atomic.AddInt32(&this.pool.running, 1)
	for {
		f.task(f.arg)
		if this.pool.push(this) != nil {
			return
		}
		if f = <-this.itm; f.task == nil {
			return
		}
	}
}
