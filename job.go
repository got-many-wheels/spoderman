package main

import "sync"

type job struct {
	url   string
	depth int
}

type jobQueue struct {
	cond   *sync.Cond
	mu     sync.Mutex
	queue  []job
	closed bool
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

	for _, j := range jobs {
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

func (jq *jobQueue) close() {
	jq.mu.Lock()
	defer jq.mu.Unlock()
	jq.closed = true
	jq.cond.Broadcast()
}
