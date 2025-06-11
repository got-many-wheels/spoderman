package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/net/html"
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
	flag.Parse()
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

	var jwg sync.WaitGroup
	jobs := make(chan job, len(lines))

	var wg sync.WaitGroup
	for i := 0; i < workersCount; i++ {
		wg.Add(1)
		go execute(pool, jobs, &wg, &jwg)
	}

	// set initial jobs from the given urls
	for _, url := range lines {
		jwg.Add(1)
		jobs <- job{url: url, depth: 0}
	}

	jwg.Wait()
	fmt.Println("All jobs finished, closing jobs channel.")
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
			return []string{}, err
		}
		// make sure to split line by whitespace if there's any
		ret = append(ret, strings.Fields(line)...)
		if !fromPipe {
			break
		}
	}

	return slices.Compact(ret), nil
}

func req(url string, buf *[]byte) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http response error: %v", resp.Status)
	}
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	*buf = append(*buf, payload...)
	return nil
}

func getUrls(payload []byte) []string {
	urls := []string{}
	tokenizer := html.NewTokenizer(bytes.NewReader(payload))
	for {
		tok := tokenizer.Next()
		switch tok {
		case html.ErrorToken:
			return urls
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			if token.Data == "a" {
				for _, attr := range token.Attr {
					if attr.Key == "href" && strings.HasPrefix(attr.Val, "http") {
						urls = append(urls, attr.Val)
					}
				}
			}
		}
	}
}

func execute(pool *sync.Pool, jobs chan job, wg *sync.WaitGroup, jwg *sync.WaitGroup) {
	defer wg.Done()
	for j := range jobs {
		buf := pool.Get().([]byte)[:0]
		if j.depth == maxDepth {
			jwg.Done()
			pool.Put(buf)
			continue
		}
		err := req(j.url, &buf)
		if err != nil {
			fmt.Printf("[X] Error while requesting to %v\n", j.url)
			jwg.Done()
			pool.Put(buf)
			continue
		}
		urls := getUrls(buf)
		for _, url := range urls {
			jwg.Add(1)
			go func() {
				jobs <- job{url: url, depth: j.depth + 1}
				fmt.Println("[+]", url, j.depth+1)
			}()
		}
		pool.Put(buf)
		jwg.Done()
	}
}
