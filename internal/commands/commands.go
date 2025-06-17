package commands

import (
	"bufio"
	"context"
	"errors"
	"os"
	"slices"

	"github.com/got-many-wheels/spoderman/internal/crawler"
	"github.com/got-many-wheels/spoderman/internal/logger"
	"github.com/got-many-wheels/spoderman/internal/utils"
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
			&ucli.StringFlag{
				Name:    "url",
				Value:   "",
				Usage:   "Target Url.",
				Aliases: []string{"u"},
			},
			&ucli.StringFlag{
				Name:    "url-file",
				Value:   "",
				Usage:   "Target urls file, separated by line break.",
				Aliases: []string{"f"},
			},
		},
		Action: func(ctx context.Context, c *ucli.Command) error {
			var urls []string
			fUrl, fUrlFile := c.String("url"), c.String("url-file")
			if len(fUrl) == 0 && len(fUrlFile) == 0 {
				return errors.New("Target url is required")
			} else {
				if len(fUrl) > 0 {
					if utils.IsValidUrl(fUrl) {
						urls = append(urls, fUrl)
					} else {
						return errors.New("Url is not valid")
					}
				}

				if len(fUrlFile) > 0 {
					f, err := os.Open(fUrlFile)
					if err != nil {
						return err
					}
					defer f.Close()
					scanner := bufio.NewScanner(f)
					for scanner.Scan() {
						txt := scanner.Text()
						if utils.IsValidUrl(txt) {
							urls = append(urls, txt)
						}
					}
				}
			}

			crawler := crawler.New(
				logger,
				slices.Compact(urls),
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
