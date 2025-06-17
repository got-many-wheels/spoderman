package crawler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/got-many-wheels/spoderman/internal/logger"
	"golang.org/x/net/html"
)

type Crawler struct {
	urls   []string
	logger *logger.Logger
	config Config
	wg     sync.WaitGroup
	jq     *jobQueue
}

type Config struct {
	Workers int
	Depth   int
	Base    bool
}

func New(logger *logger.Logger, urls []string, c Config) *Crawler {
	return &Crawler{
		urls:   urls,
		logger: logger,
		config: c,
		jq:     newJobQueue(),
	}
}

func (c *Crawler) Do() error {
	newNetClient()
	if len(c.urls) == 0 {
		return errors.New("Please provide at least 1 url to crawl to")
	}
	var numWorkerCreated int64
	pool := &sync.Pool{
		New: func() any {
			atomic.AddInt64(&numWorkerCreated, 1)
			buf := make([]byte, 0, 1024*32)
			return buf
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for i := 0; i < c.config.Workers; i++ {
		c.wg.Add(1)
		go c.execute(pool, ctx)
	}

	initialJobs := make([]job, 0, len(c.urls))
	for _, initialUrl := range c.urls {
		if c.config.Base {
			u, err := url.Parse(initialUrl)
			if err != nil {
				c.logger.Debug().Err(err).Msg(fmt.Sprintf("Error while parsing initial url %v\n", initialUrl))
				continue
			}
			hostname := u.Hostname()
			c.jq.basePaths.Store(hostname, true)
		}
		initialJobs = append(initialJobs, job{url: initialUrl, depth: 1})
	}
	c.jq.enqueue(initialJobs)

	go func() {
		<-sigChan
		c.logger.Info().Msg("Received shutdown signal, initiating graceful shutdown...")
		cancel()
		c.jq.clearJobWaitGroup() // clear all wait group in order to unblock job wait group
		c.jq.clear()
	}()

	c.jq.clear()
	c.wg.Wait()

	c.logger.Debug().Msg(fmt.Sprintf("%d worker instance created", int(numWorkerCreated)))
	c.logger.Info().Msg(fmt.Sprintf("%d links crawled successfully", c.jq.crawled))
	return nil
}

func (c *Crawler) req(url string, buf *[]byte, ctx context.Context) error {
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

func (c *Crawler) getUrls(u string, payload []byte) []string {
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

func (c *Crawler) execute(pool *sync.Pool, ctx context.Context) {
	defer c.wg.Done()
	for {
		j, ok := c.jq.dequeue()
		if !ok {
			break // queue is empty and closed
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		func() {
			buf := pool.Get().([]byte)[:0]
			defer pool.Put(buf)
			defer c.jq.jwg.Done()

			if c.config.Depth != 0 && j.depth > c.config.Depth {
				return
			}

			// do check hostname if the new url is within the initial url hostname
			if c.config.Base {
				u, err := url.Parse(j.url)
				if err != nil {
					c.logger.Debug().Err(err).Msg(fmt.Sprintf("Error while parsing job url %v\n", j.url))
					return
				}
				hostname := u.Hostname()
				_, present := c.jq.basePaths.Load(hostname)
				if !present {
					return
				}
			}

			c.logger.Debug().Msg(fmt.Sprintf("Visiting %s", j.url))

			err := c.req(j.url, &buf, ctx)
			if err != nil {
				// ignore expected canceled error
				if errors.Is(err, context.Canceled) {
					return
				}
				c.logger.Debug().Err(err).Msg(fmt.Sprintf("Error while requesting to %v\n", j.url))
				return
			}

			urls := c.getUrls(j.url, buf)

			if len(urls) > 0 {
				newJobs := make([]job, 0, len(urls))
				for _, url := range urls {
					if c.jq.isVisited(url) {
						continue
					}
					newJobs = append(newJobs, job{url: url, depth: j.depth + 1})
				}
				select {
				case <-ctx.Done():
					return
				default:
					if len(newJobs) > 0 {
						c.jq.enqueue(newJobs)
					}
				}
			}
		}()
	}
}
