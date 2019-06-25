package gpool

import (
	"context"
	"sync"
	"testing"
	"time"
)

const (
	runCnt        = 1000000
	benchParam    = 10
	benchPoolSize = 200000
)

func demoPoolFunc(args interface{}) {
	time.Sleep(time.Duration(args.(int)) * time.Millisecond)
}

func BenchmarkGoroutine(b *testing.B) {
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(runCnt)
		for j := 0; j < runCnt; j++ {
			go func() {
				demoPoolFunc(benchParam)
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkPool(b *testing.B) {
	var wg sync.WaitGroup
	p := New(context.Background(), &Config{benchPoolSize, time.Second * 100})
	defer p.Close()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(runCnt)
		for j := 0; j < runCnt; j++ {
			_ = p.Submit(func(context.Context) {
				demoPoolFunc(i)
				wg.Done()
			})
		}
		wg.Wait()
	}
	b.StopTimer()
}

func BenchmarkGoroutineUnlimit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for j := 0; j < runCnt; j++ {
			go demoPoolFunc(benchParam)
		}
	}
}

func BenchmarkPoolUnlimit(b *testing.B) {
	p := New(context.Background(), &Config{benchPoolSize, time.Second * 100})
	defer p.Close()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < runCnt; j++ {
			_ = p.Submit(func(context.Context) {
				demoPoolFunc(i)
			})
		}
	}
	b.StopTimer()
}
