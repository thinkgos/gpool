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

package gpool

import (
	"time"
)

type goWork struct {
	// pool who owns this worker.
	pool *Pool
	// task is user's task
	task chan Task
	// markTime mark when coroutine begin in idle
	markTime time.Time
}

// idleQueue implement with slice
type idleQueue struct {
	headPos int
	head    []*goWork
	tail    []*goWork
}

func NewQuickQueue() *idleQueue {
	return new(idleQueue)
}

// Len returns the length of this queue.
func (sf *idleQueue) len() int { return len(sf.head) - sf.headPos + len(sf.tail) }

// Add items to the queue
func (sf *idleQueue) insert(v *goWork) { sf.tail = append(sf.tail, v) }

// Poll retrieves and removes the head of the this Queue, or return nil if this Queue is empty.
func (sf *idleQueue) poll() *goWork {
	if sf.headPos >= len(sf.head) {
		if len(sf.tail) == 0 {
			return nil
		}
		// Pick up tail as new head, clear tail.
		sf.head, sf.headPos, sf.tail = sf.tail, 0, sf.head[:0]
	}
	v := sf.head[sf.headPos]
	sf.head[sf.headPos] = nil // should set nil for gc
	sf.headPos++
	return v
}

func (sf *idleQueue) retrieveExpiry(survival time.Duration) {
	f := func() {
		now := time.Now()
		for i := sf.headPos; i < len(sf.head); i++ {
			if now.Sub(sf.head[i].markTime) < survival {
				break
			}
			sf.head[i].task <- nil
			sf.headPos++
		}
	}
	f()
	if sf.headPos >= len(sf.head) {
		// Pick up tail as new head, clear tail.
		sf.head, sf.headPos, sf.tail = sf.tail, 0, sf.head[:0]
		f()
	}
}

func (sf *idleQueue) reset() {
	for i := sf.headPos; i < len(sf.head); i++ {
		sf.head[i].task <- nil
	}

	for i := 0; i < len(sf.tail); i++ {
		sf.tail[i].task <- nil
	}
	sf.head, sf.tail, sf.headPos = nil, nil, 0
}
