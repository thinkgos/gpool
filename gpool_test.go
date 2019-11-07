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
	"testing"
	"time"
)

const (
	benchRunCnt  = 1000000
	benchParam   = 10
	benchPoolCap = 200000
)

func poolFunc() {
	time.Sleep(benchParam * time.Millisecond)
}

func BenchmarkGoroutineUnlimit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for j := 0; j < benchRunCnt; j++ {
			go poolFunc()
		}
	}
}

func BenchmarkPoolUnlimit(b *testing.B) {
	p := New(Config{benchPoolCap, time.Second * 10, time.Second * 10})
	defer p.CloseGrace()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < benchRunCnt; j++ {
			_ = p.Submit(poolFunc)
		}
	}
	b.StopTimer()
}

func TestNewWithConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		p := New()
		defer p.CloseGrace()
		if p.Cap() != DefaultCapacity {
			t.Errorf("Pool.Cap() = %v, want %v", p.Cap(), DefaultCapacity)
		}
		if p.Len() != 0 {
			t.Errorf("Pool.Len() = %v, want %v", p.Len(), 0)
		}
		if p.Free() != DefaultCapacity {
			t.Errorf("Pool.Free() = %v, want %v", p.Free(), DefaultCapacity)
		}
		if p.Idle() != 0 {
			t.Errorf("Pool.Idle() = %v, want %v", p.Idle(), 0)
		}
	})

	t.Run("invalid config cap use default", func(t *testing.T) {
		p := New(Config{-1, time.Second * 1, DefaultMiniCleanupTime})
		defer p.CloseGrace()
		if p.Cap() != DefaultCapacity {
			t.Errorf("Pool.Cap() = %v, want %v", p.Cap(), DefaultCapacity)
		}
		if p.Len() != 0 {
			t.Errorf("Pool.Len() = %v, want %v", p.Len(), 0)
		}
		if p.Free() != DefaultCapacity {
			t.Errorf("Pool.Free() = %v, want %v", p.Free(), DefaultCapacity)
		}
	})

	t.Run("use user config", func(t *testing.T) {
		want := 10000
		p := New(Config{want, time.Second * 1, DefaultMiniCleanupTime})
		defer p.CloseGrace()
		if p.Cap() != want {
			t.Errorf("Pool.Cap() = %v, want %v", p.Cap(), want)
		}
		if p.Len() != 0 {
			t.Errorf("Pool.Len() = %v, want %v", p.Len(), 0)
		}
		if p.Free() != want {
			t.Errorf("Pool.Free() = %v, want %v", p.Free(), want)
		}
	})

}

func TestWithWork(t *testing.T) {
	t.Run("invalid function task", func(t *testing.T) {
		p := New()
		defer p.CloseGrace()
		err := p.Submit(nil)
		if err == nil {
			t.Errorf("Pool.Submit() Err = %v, want %v", err, ErrInvalidFunc)
		}
	})

	t.Run("do task when pool is closed", func(t *testing.T) {
		p := New()
		p.CloseGrace()
		time.Sleep(200 * time.Millisecond)
		err := p.Submit(poolFunc)
		if err == nil {
			t.Errorf("Pool.Submit() Err = %v, want %v", err, ErrClosed)
		}
	})

	t.Run("check pool parameters", func(t *testing.T) {
		p := New()
		defer p.CloseGrace()
		err := p.Submit(poolFunc)
		if err != nil {
			t.Errorf("Pool.Submit() Err = %v, want %v", err, nil)
		}
		_ = p.Submit(poolFunc)
		_ = p.Submit(poolFunc)
		if p.Cap() != DefaultCapacity {
			t.Errorf("Pool.Cap() = %v, want %v", p.Cap(), DefaultCapacity)
		}
		if p.Len() != 3 {
			t.Errorf("Pool.Len() = %v, want %v", p.Len(), 3)
		}
		if p.Free() != DefaultCapacity-3 {
			t.Errorf("Pool.Free() = %v, want %v", p.Free(), DefaultCapacity-3)
		}
		if p.Idle() != 0 {
			t.Errorf("Pool.Idle() = %v, want %v", p.Idle(), 0)
		}

		t.Log("task done then pool collect idle goroutine")
		time.Sleep(11 * time.Millisecond)
		if p.Idle() != 3 {
			t.Errorf("Pool.Idle() = %v, want %v", p.Idle(), 3)
		}
		t.Log("all goroutine done")
		time.Sleep(time.Second * 3)
		if p.Len() != 0 {
			t.Errorf("Pool.Len() = %v, want %v", p.Len(), 0)
		}
		if p.Free() != DefaultCapacity {
			t.Errorf("Pool.Free() = %v, want %v", p.Free(), DefaultCapacity)
		}
		if p.Idle() != 0 {
			t.Errorf("Pool.Idle() = %v, want %v", p.Idle(), 0)
		}

		p.Adjust(20000)
		if p.Cap() != 20000 {
			t.Errorf("after Pool.Adjust, Pool.Cap() = %v, want %v", p.Idle(), 20000)
		}

		p.Adjust(-1) // just for coverage
	})

	t.Run("close by user", func(t *testing.T) {
		p := New()
		_ = p.Submit(poolFunc)
		_ = p.Submit(poolFunc)
		time.Sleep(time.Millisecond * 2)
		_ = p.Submit(poolFunc)
		p.CloseGrace()
		p.CloseGrace() // close twice
		t.Log("all goroutine done")
		time.Sleep(time.Millisecond * 100)
		if p.Len() != 0 {
			t.Errorf("Pool.Len() = %v, want %v", p.Len(), 0)
		}
		if p.Free() != DefaultCapacity {
			t.Errorf("Pool.Free() = %v, want %v", p.Free(), DefaultCapacity)
		}
		if p.Idle() != 0 {
			t.Errorf("Pool.Idle() = %v, want %v", p.Idle(), 0)
		}
	})
}

func TestWithFullWork(t *testing.T) {
	p := New(Config{5, time.Second * 1, DefaultMiniCleanupTime})
	defer p.CloseGrace()
	for i := 0; i < 10; i++ {
		_ = p.Submit(poolFunc)
	}
	t.Log("pool full then wait for idle goroutine")
}

func TestWithWorkPanic(t *testing.T) {
	p := New()
	defer p.CloseGrace()
	p.SetPanicHandler(func() {
		t.Log("panic happen")
	})

	_ = p.Submit(func() {
		panic("painc happen")
	})
	time.Sleep(time.Second * 1)
}
