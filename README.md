## gpool
goroutine pool package

NOTE: archived use [ants](https://github.com/panjf2000/ants) instead.


[![GoDoc](https://godoc.org/github.com/thinkgos/gpool?status.svg)](https://godoc.org/github.com/thinkgos/gpool)
[![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-blue?logo=go&logoColor=white)](https://pkg.go.dev/github.com/thinkgos/gpool?tab=doc)
[![Build Status](https://www.travis-ci.org/thinkgos/gpool.svg?branch=master)](https://www.travis-ci.org/thinkgos/gpool)
[![codecov](https://codecov.io/gh/thinkgos/gpool/branch/master/graph/badge.svg)](https://codecov.io/gh/thinkgos/gpool)
![Action Status](https://github.com/thinkgos/gpool/workflows/Go/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/thinkgos/gpool)](https://goreportcard.com/report/github.com/thinkgos/gpool)
[![License](https://img.shields.io/github/license/thinkgos/gpool)](https://github.com/thinkgos/gpool/raw/master/LICENSE)
[![Tag](https://img.shields.io/github/v/tag/thinkgos/gpool)](https://github.com/thinkgos/gpool/tags)


## 说明
 - 原来是采用单chan 实现协程池,然后发现性能瓶颈在上万个goroutine在竞争同一个chan,严重影响并发性,
 - 改为链表并排序,回收活动但无任务的协程的放在链尾,取任务时取链头,以保证协程池有一定的活动数量.并且每个链表节点都有分配一个为1的chan.性能比单chan提高40%
 - 压力测试条件,电脑4核8G内存ubuntu18.04系统,100W并发任务,每个任务执行10ms,协程池20W个
   ```
   原生:   BenchmarkGoroutineUnlimit-8   	       2	 588321422 ns/op	80036632 B/op	 1000365 allocs/op
   协程池: BenchmarkPoolUnlimit-8   	       2	 746699636 ns/op	 4050860 B/op	   50196 allocs/op
   ```
 - 压力测试条件,电脑4核8G内存ubuntu18.04系统,100W并发任务,每个任务执行10ms,协程池1W个
   ```
   原生:   BenchmarkGoroutineUnlimit-8   	       3	 547231063 ns/op	80001594 B/op	 1000011 allocs/op
   协程池: BenchmarkPoolUnlimit-8   	       1	1116941925 ns/op	 8656656 B/op	   61128 allocs/op
   ```
 - 发现有个开源方案,用原生slice实现的,功能更全,优化更足,建议使用.
   see: [ants](https://github.com/panjf2000/ants)