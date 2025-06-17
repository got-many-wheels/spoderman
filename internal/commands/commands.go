package commands

import (
	"context"

	"github.com/got-many-wheels/spoderman/internal/crawler"
	"github.com/got-many-wheels/spoderman/internal/logger"
	ucli "github.com/urfave/cli/v3"
)

func Get(logger *logger.Logger) []*ucli.Command {
	return []*ucli.Command{
		Crawl(logger),
	}
}

func Crawl(logger *logger.Logger) *ucli.Command {
	cmd := &ucli.Command{
		Name:  "crawl",
		Usage: "Start the crawling process",
		Flags: []ucli.Flag{
			&ucli.IntFlag{
				Name:    "depth",
				Value:   1,
				Aliases: []string{"d"},
				Usage:   "Maximum depth for crawling. Higher values crawl deeper into link trees.",
			},
			&ucli.IntFlag{
				Name:    "workers",
				Value:   10,
				Aliases: []string{"w"},
				Usage:   "Number of concurrent workers to crawl URLs in parallel.",
			},
			&ucli.BoolFlag{
				Name:    "base",
				Value:   true,
				Usage:   "Restrict crawling to the base domain only.",
				Aliases: []string{"b"},
			},
		},
		Action: func(ctx context.Context, c *ucli.Command) error {
			crawler := crawler.New(
				logger,
				crawler.Config{
					Workers: c.Int("workers"),
					Depth:   c.Int("depth"),
					Base:    c.Bool("base"),
				},
			)
			return crawler.Do()
		},
	}
	return cmd
}
