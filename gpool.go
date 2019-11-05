// MIT License
//
// Copyright (c) 2019 jiang
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Package gpool Implementing a goroutine pool
package gpool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// default config parameter
const (
	DefaultCapacity        = 100000
	DefaultSurvivalTime    = 1 * time.Second
	DefaultMiniCleanupTime = 100 * time.Millisecond
)

// define pool state
const (
	onWork = iota
	closed
)

var (
	// ErrClosed indicate the pool has closed
	ErrClosed = errors.New("pool has closed")
	// ErrInvalidFunc indicate the task function is invalid
	ErrInvalidFunc = errors.New("invalid function, must not be nil")
	// ErrOverload indicate the goroutine overload
	ErrOverload = errors.New("pool overload")
)

// TaskFunc task function define
type TaskFunc func(arg interface{})

// Config the pool config parameter
type Config struct {
	Capacity        int
	SurvivalTime    time.Duration
	MiniCleanupTime time.Duration // mini cleanup time
}

// Pool the goroutine pool
type Pool struct {
	ctx    context.Context
	cancel context.CancelFunc

	capacity        int32 // goroutines capacity
	running         int32 // goroutines running count
	survivalTime    time.Duration
	miniCleanupTime time.Duration // mini cleanup time

	closeDone uint32

	mux            sync.Mutex
	cond           *sync.Cond
	idleGoRoutines *list // idle go routine list
	cache          *sync.Pool
	wg             sync.WaitGroup

	panicFunc func()
}

// New new a pool with the config if there is ,other use default config
func New(c ...Config) *Pool {
	if len(c) == 0 {
		c = append(c, Config{
			Capacity:        DefaultCapacity,
			SurvivalTime:    DefaultSurvivalTime,
			MiniCleanupTime: DefaultMiniCleanupTime,
		})
	}
	if c[0].Capacity < 0 {
		c[0].Capacity = DefaultCapacity
	}
	if c[0].MiniCleanupTime < DefaultMiniCleanupTime {
		c[0].MiniCleanupTime = DefaultMiniCleanupTime
	}

	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		ctx:    ctx,
		cancel: cancel,

		capacity:        int32(c[0].Capacity),
		survivalTime:    c[0].SurvivalTime,
		miniCleanupTime: c[0].MiniCleanupTime,

		idleGoRoutines: newList(),
	}
	p.cond = sync.NewCond(&p.mux)
	p.cache = &sync.Pool{
		New: func() interface{} { return &work{itm: make(chan item, 1), pool: p} },
	}
	go p.cleanUp()
	return p
}

func (sf *Pool) cleanUp() {
	tick := time.NewTimer(sf.survivalTime)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			nearTimeout := sf.survivalTime
			now := time.Now()
			sf.mux.Lock()
			var next *work
			for e := sf.idleGoRoutines.Front(); e != nil; e = next {
				if nearTimeout = now.Sub(e.markTime); nearTimeout < sf.survivalTime {
					break
				}
				next = e.Next() // save before delete
				sf.idleGoRoutines.remove(e).itm <- item{}
			}
			sf.mux.Unlock()
			if nearTimeout < sf.miniCleanupTime {
				nearTimeout = sf.miniCleanupTime
			}
			tick.Reset(nearTimeout)
		case <-sf.ctx.Done():
			sf.mux.Lock()
			for e := sf.idleGoRoutines.Front(); e != nil; e = e.Next() {
				e.itm <- item{} // give a nil function, make all goroutine exit
			}
			sf.idleGoRoutines = nil
			sf.mux.Unlock()
			return
		}
	}
}

// SetPanicHandler set panic handler
func (sf *Pool) SetPanicHandler(f func()) {
	sf.panicFunc = f
}

// Len returns the currently running goroutines
func (sf *Pool) Len() int {
	return int(atomic.LoadInt32(&sf.running))
}

// Cap tha capacity of goroutines the pool can create
func (sf *Pool) Cap() int {
	return int(atomic.LoadInt32(&sf.capacity))
}

// Adjust adjust the capacity of the pools goroutines
func (sf *Pool) Adjust(size int) {
	if size < 0 || sf.Cap() == size {
		return
	}
	atomic.StoreInt32(&sf.capacity, int32(size))
}

// Free return the available goroutines can create
func (sf *Pool) Free() int {
	return sf.Cap() - sf.Len()
}

// Idle return the goroutines has running but in idle(no task work)
func (sf *Pool) Idle() int {
	var cnt int
	sf.mux.Lock()
	if sf.idleGoRoutines != nil {
		cnt = sf.idleGoRoutines.Len()
	}
	sf.mux.Unlock()
	return cnt
}

// Close the pool,if grace enable util all goroutine close
func (sf *Pool) close(grace bool) error {
	if atomic.LoadUint32(&sf.closeDone) == closed {
		return nil
	}

	sf.mux.Lock()
	if sf.closeDone == onWork { // check again,make sure
		sf.cancel()
		atomic.StoreUint32(&sf.closeDone, closed)
	}
	sf.mux.Unlock()
	if grace {
		sf.wg.Wait()
	}
	return nil
}

// Close the pool,but not wait all goroutine close
func (sf *Pool) Close() error {
	return sf.close(false)
}

// CloseGrace the pool,wait util all goroutine close
func (sf *Pool) CloseGrace() error {
	return sf.close(true)
}

// Submit submits a task with arg
func (sf *Pool) Submit(f TaskFunc, arg interface{}) error {
	var w *work

	if f == nil {
		return ErrInvalidFunc
	}

	if atomic.LoadUint32(&sf.closeDone) == closed {
		return ErrClosed
	}

	sf.mux.Lock()
	if sf.closeDone == closed || sf.idleGoRoutines == nil { // check again,make sure
		sf.mux.Unlock()
		return ErrClosed
	}

	itm := item{f, arg}
	if w = sf.idleGoRoutines.Front(); w != nil {
		sf.idleGoRoutines.Remove(w)
		sf.mux.Unlock()
		w.itm <- itm
		return nil
	}

	// actual goroutines maybe greater than cap, when race, but it will overload and return to normal in goroutine
	if sf.Free() > 0 {
		sf.mux.Unlock()
		w = sf.cache.Get().(*work)
		w.run(itm)
		return nil
	}

	for {
		sf.cond.Wait()
		if w = sf.idleGoRoutines.Front(); w != nil {
			sf.idleGoRoutines.Remove(w)
			break
		}
	}
	sf.mux.Unlock()
	w.itm <- itm
	return nil
}

// push the running goroutine to idle pool
func (sf *Pool) push(w *work) error {
	if atomic.LoadUint32(&sf.closeDone) == closed { // quick check
		return ErrClosed
	}

	if sf.Free() < 0 {
		return ErrOverload
	}

	w.markTime = time.Now()
	sf.mux.Lock()
	if sf.closeDone == closed { // check again,make sure
		sf.mux.Unlock()
		return ErrClosed
	}
	sf.idleGoRoutines.PushBack(w)
	sf.cond.Signal()
	sf.mux.Unlock()
	return nil
}

func (sf *work) run(itm item) {
	sf.pool.wg.Add(1)
	atomic.AddInt32(&sf.pool.running, 1)
	go func() {
		defer func() {
			sf.pool.wg.Done()
			atomic.AddInt32(&sf.pool.running, -1)
			sf.pool.cache.Put(sf)
			if r := recover(); r != nil && sf.pool.panicFunc != nil {
				sf.pool.panicFunc()
			}
		}()

		for {
			itm.task(itm.arg)
			if sf.pool.push(sf) != nil {
				return
			}
			if itm = <-sf.itm; itm.task == nil {
				return
			}
		}
	}()
}
