package crawler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/got-many-wheels/spoderman/internal/config"
	"github.com/got-many-wheels/spoderman/internal/logger"
)

type Crawler struct {
	urls    []string
	logger  *logger.Logger
	filters *chainedFilters
	config  config.Config
	wg      sync.WaitGroup
	jq      *jobQueue
}

func New(logger *logger.Logger, urls []string, c config.Config) *Crawler {
	var f []urlFilter
	if len(c.AllowedDomains) > 0 {
		f = append(f, &allowedFilter{allowed: c.AllowedDomains})
	}
	f = append(f, &disallowedFilter{disallowed: c.DisallowedDomains})
	filters := &chainedFilters{filters: f}

	return &Crawler{
		urls:    urls,
		logger:  logger,
		config:  c,
		jq:      newJobQueue(),
		filters: filters,
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

	for i := 0; i < *c.config.Workers; i++ {
		c.wg.Add(1)
		go c.execute(pool, ctx)
	}

	initialJobs := make([]job, 0, len(c.urls))
	for _, initialUrl := range c.urls {
		if err := c.jq.storeBasePath(initialUrl); err != nil {
			c.logger.Error().Msg(err.Error())
			continue
		}
		initialJobs = append(initialJobs, job{url: initialUrl, depth: 1})
	}
	c.jq.enqueue(initialJobs, []Secret{})

	go func() {
		<-sigChan
		c.logger.Info().Msg("Received shutdown signal, initiating graceful shutdown...")
		cancel()
		c.jq.clearJobWaitGroup() // clear all wait group in order to unblock job wait group
		c.jq.clear()
	}()

	c.jq.clear()
	c.wg.Wait()
	c.jq.outputResults(c.config.Output)

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

			if *c.config.Depth != 0 && j.depth > *c.config.Depth {
				return
			}

			// do check hostname if the new url is within the initial url hostname
			u, err := url.Parse(j.url)
			if err != nil {
				c.logger.Debug().Err(err).Msg(fmt.Sprintf("Error while parsing job url %v\n", j.url))
				return
			}

			// TODO: should we keep this? since we already have domain filters
			hostname := u.Hostname()
			if *c.config.Base {
				_, present := c.jq.basePaths.Load(hostname)
				if !present {
					return
				}
			}

			c.logger.Debug().Msg(fmt.Sprintf("Visiting %s", j.url))

			err = c.req(j.url, &buf, ctx)
			if err != nil {
				// ignore expected canceled error
				if errors.Is(err, context.Canceled) {
					return
				}
				c.logger.Debug().Err(err).Msg(fmt.Sprintf("Error while requesting to %v\n", j.url))
				return
			}
			pNode := newPageNode(j.url, buf)
			if err := pNode.extractAndExtends(hostname); err != nil {
				c.logger.Debug().Err(err).Msg(fmt.Sprintf("Error while extracting html content\n"))
				return
			}

			newJobs := make([]job, 0, len(pNode.foundUrls))
			for _, url := range pNode.foundUrls {
				if c.jq.isVisited(url) {
					continue
				}
				if !c.filters.allow(url) {
					continue
				}
				newJobs = append(newJobs, job{url: url, depth: j.depth + 1})
			}
			select {
			case <-ctx.Done():
				return
			default:
				c.jq.enqueue(newJobs, pNode.foundSecrets)
			}
		}()
	}
}
