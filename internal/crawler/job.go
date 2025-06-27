package crawler

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/go-memdb"
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
	db        *memdb.MemDB
}

func newJobQueue() *jobQueue {
	jq := &jobQueue{}
	jq.cond = sync.NewCond(&jq.mu)

	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"secret": {
				Name: "secret",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "HostnameKey"},
					},
					"key": {
						Name:    "key",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "Key"},
					},
					"value": {
						Name:    "value",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "Value"},
					},
				},
			},
		},
	}

	db, err := memdb.NewMemDB(schema)
	if err != nil {
		panic(err)
	}
	jq.db = db
	return jq
}

func (jq *jobQueue) enqueue(jobs []job, foundSecrets []Secret) {
	jq.mu.Lock()
	defer jq.mu.Unlock()
	if jq.closed {
		return
	}

	txn := jq.db.Txn(true)
	for _, secret := range foundSecrets {
		if err := txn.Insert("secret", secret); err != nil {
			panic(err)
		}
	}
	txn.Commit()

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

func (jq *jobQueue) outputResults() {
	txn := jq.db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get("secret", "id")
	if err != nil {
		panic(err)
	}

	// TODO: output secret as csv or json
	for obj := it.Next(); obj != nil; obj = it.Next() {
		p := obj.(Secret)
		fmt.Printf("  %s %s\n", p.Key, p.Value)
	}
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
