package app

import (
	"context"

	"github.com/got-many-wheels/spoderman/internal/commands"
	"github.com/got-many-wheels/spoderman/internal/config"
	"github.com/got-many-wheels/spoderman/internal/logger"
	ucli "github.com/urfave/cli/v3"
)

type App struct {
	Cli    ucli.Command
	Logger *logger.Logger
	Config *config.Config
}

func New() *App {
	app := &App{
		Logger: logger.New(false),
		Config: config.New(),
	}
	app.Cli = ucli.Command{
		Name:  "spoderman",
		Usage: "Dead simple website crawler",
		Flags: []ucli.Flag{
			&ucli.BoolFlag{
				Name:  "verbose",
				Value: *app.Config.Verbose,
				Usage: "Enable verbose logging",
			},
		},
		Before: func(ctx context.Context, c *ucli.Command) (context.Context, error) {
			verbose := c.Bool("verbose")
			if verbose {
				app.Logger.ToVerbose(verbose)
			}
			return ctx, nil
		},
		Commands: commands.Get(app.Config, app.Logger),
	}
	return app
}
