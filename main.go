package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/term"
)

var (
	// flags
	maxDepth     int
	workersCount int
	verbose      bool
	base         bool

	_logger   *logger
	once      sync.Once
	netClient *http.Client
)

func init() {
	flag.IntVar(&maxDepth, "depth", 1, "Maximum depth for crawling. Higher values crawl deeper into link trees. (default: 1)")
	flag.IntVar(&workersCount, "workersCount", 10, "Number of concurrent workers to crawl URLs in parallel (default: 10)")
	flag.BoolVar(&verbose, "verbose", false, "Enables detailed logs for each crawling operation.")
	flag.BoolVar(&base, "base", false, "Restrict crawling to the base domain only (same host as initial URL)")
}

func main() {
	flag.Parse()
	newNetClient()
	_logger = newLogger(verbose)
	_logger.Debug().Int("workersCount", workersCount)
	lines, err := readLines()
	if err != nil {
		panic(err)
	}
	_logger.Debug().Int("lines", len(lines))
	var numWorkerCreated int64
	pool := &sync.Pool{
		New: func() any {
			atomic.AddInt64(&numWorkerCreated, 1)
			buf := make([]byte, 0, 1024*32)
			return buf
		},
	}

	var wg sync.WaitGroup
	jq := newJobQueue()

	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for i := 0; i < workersCount; i++ {
		wg.Add(1)
		go execute(pool, jq, &wg, ctx)
	}

	// set initial jobs from the given urls
	initialJobs := make([]job, 0, len(lines))
	for _, initialUrl := range lines {
		if base {
			u, err := url.Parse(initialUrl)
			if err != nil {
				_logger.log.Debug().Err(err).Msg(fmt.Sprintf("Error while parsing initial url %v\n", initialUrl))
				continue
			}
			hostname := u.Hostname()
			jq.basePaths.Store(hostname, true)
		}
		initialJobs = append(initialJobs, job{url: initialUrl, depth: 0})
	}
	jq.enqueue(initialJobs)

	go func() {
		<-sigChan
		_logger.Info().Msg("Received shutdown signal, initiating graceful shutdown...")
		cancel()
		jq.clearJobWaitGroup() // clear all wait group in order to unblock job wait group
		jq.clear()
	}()

	jq.clear()
	wg.Wait()

	_logger.Debug().Msg(fmt.Sprintf("%d worker instance created", int(numWorkerCreated)))
	_logger.Info().Msg(fmt.Sprintf("%d links crawled successfully", jq.crawled))
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

func newNetClient() *http.Client {
	once.Do(func() {
		var netTransport = &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 2 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 2 * time.Second,
		}
		netClient = &http.Client{
			Timeout:   time.Second * 2,
			Transport: netTransport,
		}
	})
	return netClient
}

func req(url string, buf *[]byte, ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := netClient.Do(req)
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

func getUrls(u string, payload []byte) []string {
	urls := []string{}
	tokenizer := html.NewTokenizer(bytes.NewReader(payload))
	baseURL, err := url.Parse(u)
	if err != nil {
		return urls
	}
	for {
		tok := tokenizer.Next()
		switch tok {
		case html.ErrorToken:
			return urls
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			if token.Data == "a" {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						href := strings.TrimSpace(attr.Val)
						parsedHref, err := url.Parse(href)
						if err != nil {
							continue
						}
						fullURL := baseURL.ResolveReference(parsedHref).String()
						urls = append(urls, fullURL)
					}
				}
			}
		}
	}
}

func execute(pool *sync.Pool, jq *jobQueue, wg *sync.WaitGroup, ctx context.Context) {
	defer wg.Done()
	for {
		buf := pool.Get().([]byte)[:0]
		j, ok := jq.dequeue()
		if !ok {
			pool.Put(buf)
			break // queue is empty and closed
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		func() {
			defer pool.Put(buf)
			defer jq.jwg.Done()

			if j.depth == maxDepth {
				return
			}

			// do check hostname if the new url is within the initial url hostname
			if base {
				u, err := url.Parse(j.url)
				if err != nil {
					_logger.log.Debug().Err(err).Msg(fmt.Sprintf("Error while parsing job url %v\n", j.url))
					return
				}
				hostname := u.Hostname()
				_, present := jq.basePaths.Load(hostname)
				if !present {
					return
				}
			}

			_logger.log.Debug().Msg(fmt.Sprintf("%s", j.url))

			err := req(j.url, &buf, ctx)
			if err != nil {
				// ignore expected canceled error
				if errors.Is(err, context.Canceled) {
					return
				}
				_logger.log.Debug().Err(err).Msg(fmt.Sprintf("Error while requesting to %v\n", j.url))
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
			}

			urls := getUrls(j.url, buf)

			if len(urls) > 0 {
				newJobs := make([]job, 0, len(urls))
				for _, url := range urls {
					if jq.isVisited(url) {
						continue
					}
					newJobs = append(newJobs, job{url: url, depth: j.depth + 1})
				}
				select {
				case <-ctx.Done():
					return
				default:
					if len(newJobs) > 0 {
						jq.enqueue(newJobs)
					}
				}
			}
		}()
	}
}
