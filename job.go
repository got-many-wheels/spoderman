package main

import (
	"sync"
	"sync/atomic"
)

type job struct {
	url   string
	depth int
}

type jobQueue struct {
	cond      *sync.Cond
	mu        sync.Mutex
	queue     []job
	closed    bool
	crawled   int64 // count of successful crawled urls
	basePaths sync.Map
	sm        sync.Map
	jwg       sync.WaitGroup // jobs wait group
}

func newJobQueue() *jobQueue {
	jq := &jobQueue{}
	jq.cond = sync.NewCond(&jq.mu)
	return jq
}

func (jq *jobQueue) enqueue(jobs []job) {
	jq.mu.Lock()
	defer jq.mu.Unlock()
	if jq.closed {
		return
	}
	atomic.AddInt64(&jq.crawled, int64(len(jobs)))
	for _, j := range jobs {
		jq.jwg.Add(1)
		jq.queue = append(jq.queue, j)
	}
	jq.cond.Broadcast()
}

func (jq *jobQueue) dequeue() (job, bool) {
	jq.mu.Lock()
	defer jq.mu.Unlock()

	// wait until there's work to do, or else turu
	for len(jq.queue) == 0 && !jq.closed {
		jq.cond.Wait()
	}

	if jq.closed && len(jq.queue) == 0 {
		return job{}, false
	}

	job := jq.queue[0]
	jq.queue = jq.queue[1:]
	return job, true
}

func (jq *jobQueue) clear() {
	jq.jwg.Wait()
	jq.close()
}

func (jq *jobQueue) clearJobWaitGroup() {
	for range jq.queue {
		jq.jwg.Done()
	}
}

func (jq *jobQueue) close() {
	jq.mu.Lock()
	defer jq.mu.Unlock()
	jq.closed = true
	jq.cond.Broadcast()
}

func (jq *jobQueue) isVisited(u string) bool {
	_, present := jq.sm.Load(u)
	if present {
		return true
	}
	jq.sm.Store(u, true)
	return false
}
