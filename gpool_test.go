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

func poolFunc(args int) {
	time.Sleep(time.Duration(args) * time.Millisecond)
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
	defer p.Close()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < benchRunCnt; j++ {
			_ = p.Submit(func(arg interface{}) {
				poolFunc(arg.(int))
			}, benchParam)
		}
	}
	b.StopTimer()
}
