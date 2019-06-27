package gpool

import (
	"testing"
	"time"
)

const (
	benchRunCnt  = 1000000
	benchParam   = 10
	benchPoolCap = 50000
)

func poolFunc(args interface{}) {
	time.Sleep(time.Duration(args.(int)) * time.Millisecond)
}

func BenchmarkGoroutineUnlimit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for j := 0; j < benchRunCnt; j++ {
			go poolFunc(benchParam)
		}
	}
}

func BenchmarkPoolUnlimit(b *testing.B) {
	p := New(&Config{benchPoolCap, time.Second * 1})
	defer p.Close(true)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < benchRunCnt; j++ {
			_ = p.Submit(func(arg interface{}) {
				poolFunc(arg)
			}, benchParam)
		}
	}
	b.StopTimer()
}

func TestNewWithConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		p := New()
		defer p.Close(true)
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
		p := New(&Config{-1, time.Second * 1})
		defer p.Close(true)
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
		p := New(&Config{want, time.Second * 1})
		defer p.Close(true)
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
		defer p.Close(true)
		err := p.Submit(nil, 1)
		if err == nil {
			t.Errorf("Pool.Submit() Err = %v, want %v", err, ErrInvalidFunc)
		}
	})

	t.Run("do task when pool is closed", func(t *testing.T) {
		p := New()
		p.Close(true)
		time.Sleep(200 * time.Millisecond)
		err := p.Submit(poolFunc, 1)
		if err == nil {
			t.Errorf("Pool.Submit() Err = %v, want %v", err, ErrClosed)
		}
	})

	t.Run("check pool parameters", func(t *testing.T) {
		p := New()
		defer p.Close(true)
		err := p.Submit(poolFunc, 1)
		if err != nil {
			t.Errorf("Pool.Submit() Err = %v, want %v", err, nil)
		}
		_ = p.Submit(poolFunc, 1)
		_ = p.Submit(poolFunc, 1)
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
		time.Sleep(10 * time.Millisecond)
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
	})

	t.Run("close by user", func(t *testing.T) {
		p := New()

		_ = p.Submit(poolFunc, 1)
		_ = p.Submit(poolFunc, 1)
		time.Sleep(time.Millisecond * 2)
		_ = p.Submit(poolFunc, 1)
		p.Close(true)
		p.Close(true) // close twice
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
	p := New(&Config{5, time.Second * 1})
	defer p.Close(true)
	for i := 0; i < 10; i++ {
		_ = p.Submit(poolFunc, 1)
	}
	t.Log("pool full then wait for idle goroutine")
}

func TestWithWorkPanic(t *testing.T) {
	p := New()
	defer p.Close(true)
	p.SetPanicHandler(func() {
		t.Log("panic happen")
	})

	_ = p.Submit(func(interface{}) {
		panic("painc happen")
	}, 1)
	time.Sleep(time.Second * 1)
}
