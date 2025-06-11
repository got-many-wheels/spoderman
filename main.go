package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/term"
)

type job struct {
	url   string
	depth int
}

const N_WORKERS = 10

var (
	urls         []string
	maxDepth     int
	workersCount int
)

func init() {
	flag.IntVar(&maxDepth, "depth", 1, "maximum depth for the crawler (default: 1)")
	flag.IntVar(&workersCount, "workersCount", N_WORKERS, "amount of pooled worker (default: 10)")
}

func main() {
	lines, err := readLines()
	if err != nil {
		panic(err)
	}

	var numWorkerCreated int64
	pool := &sync.Pool{
		New: func() any {
			atomic.AddInt64(&numWorkerCreated, 1)
			// TODO: buffer size should be configurable?
			buf := make([]byte, 0, 1024*32)
			return buf
		},
	}

	jobs := make(chan job, len(lines))
	var wg sync.WaitGroup
	for i := 0; i < workersCount; i++ {
		wg.Add(1)
		go execute(i, pool, jobs, &wg)
	}

	for _, url := range lines {
		jobs <- job{url: url, depth: 0}
	}

	close(jobs)
	wg.Wait()
	fmt.Printf("Worker instance created: %d\n", numWorkerCreated)
}

func readLines() ([]string, error) {
	ret := []string{}
	// to check whether the input came from pipe or manual user input
	fromPipe := !term.IsTerminal(int(os.Stdin.Fd()))
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			// prevent losing trailing input on EOF
			if len(line) > 0 {
				ret = append(ret, strings.TrimSpace(line))
			}
			break
		} else if err != nil {
			return ret, err
		}
		// make sure to split line by whitespace if there's any
		ret = append(ret, strings.Fields(line)...)
		if !fromPipe {
			break
		}
	}
	return ret, nil
}

func execute(id int, pool *sync.Pool, jobs <-chan job, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		buf := pool.Get().([]byte)[:0]
		if job.depth > maxDepth {
			break
		}
		fmt.Printf("Worker %d started crawling the web\n", id)
		fmt.Println("proccessing", job.url, len(buf))
		pool.Put(buf)
	}
}
