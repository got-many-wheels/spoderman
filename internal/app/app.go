package app

import (
	"context"

	"github.com/got-many-wheels/spoderman/internal/commands"
	"github.com/got-many-wheels/spoderman/internal/logger"
	ucli "github.com/urfave/cli/v3"
)

type App struct {
	Cli    ucli.Command
	Logger *logger.Logger
}

func New() *App {
	app := &App{
		Logger: logger.New(false),
	}
	app.Cli = ucli.Command{
		Name:  "spoderman",
		Usage: "Dead simple website crawler",
		Flags: []ucli.Flag{
			&ucli.BoolFlag{
				Name:  "verbose",
				Usage: "Enable verbose logging",
			},
		},
		Before: func(ctx context.Context, c *ucli.Command) (context.Context, error) {
			verbose := c.Bool("verbose")
			if verbose {
				app.Logger.ToVerbose()
			}
			return ctx, nil
		},
		Commands: commands.Get(app.Logger),
	}
	return app
}
