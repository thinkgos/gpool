[![GoDoc](https://godoc.org/github.com/thinkgos/gpool?status.svg)](https://godoc.org/github.com/thinkgos/gpool)
[![Build Status](https://travis-ci.org/thinkgos/gpool.svg?branch=master)](https://travis-ci.org/thinkgos/gpool)
[![codecov](https://codecov.io/gh/thinkgos/gpool/branch/master/graph/badge.svg)](https://codecov.io/gh/thinkgos/gpool)
![Action Status](https://github.com/thinkgos/gpool/workflows/Go/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/thinkgos/gpool)](https://goreportcard.com/report/github.com/thinkgos/gpool)
[![Licence](https://img.shields.io/github/license/thinkgos/gpool)](https://raw.githubusercontent.com/thinkgos/gpool/master/LICENSE)
## gpool
goroutine pool package

## 说明
 - 原来是采用单chan 实现协程池,然后发现性能瓶颈在上万个goroutine在竞争同一个chan,严重影响并发性,
 - 改为链表并排序,回收活动但无任务的协程的放在链尾,取任务时取链头,以保证协程池有一定的活动数量.并且每个链表节点都有分配一个为1的chan.性能比单chan提高40%
 - 压力测试条件,电脑4核8G内存ubuntu18.04系统,100W并发任务,每个任务执行10ms,协程池5W个
   ```
   原生:   BenchmarkGoroutineUnlimit-8   	       1	1638630994 ns/op	536366136 B/op	 1999667 allocs/op
   协程池: BenchmarkPoolUnlimit-8   	       2	 818826478 ns/op	 3339796 B/op	   53062 allocs/op
   ```
 - 压力测试条件,电脑4核8G内存ubuntu18.04系统,100W并发任务,每个任务执行10ms,协程池1W个
   ```
   原生:   BenchmarkGoroutineUnlimit-8   	       1	1799574445 ns/op	536508856 B/op	 1992523 allocs/op
   协程池: BenchmarkPoolUnlimit-8   	       1	1057243165 ns/op	 8522016 B/op	   68919 allocs/op
   ```
 - 发现有个开源方案,用原生slice实现的,功能更全  
    see: [ants](https://github.com/panjf2000/ants)