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
			&ucli.StringFlag{
				Name:    "config",
				Value:   "",
				Usage:   "Set config file, defaults to set flag values or empty",
				Aliases: []string{"i"},
			},
		},
		Before: func(ctx context.Context, c *ucli.Command) (context.Context, error) {
			verbose := c.Bool("verbose")
			if verbose {
				app.Logger.ToVerbose(verbose)
			}
			cfgSrc := c.String("config")
			if len(cfgSrc) > 0 {
				cfg, err := config.ParseJsonConfig(cfgSrc)
				if err != nil {
					return ctx, err
				}
				app.Config = cfg
				app.Logger.ToVerbose(*app.Config.Verbose)
			}

			return ctx, nil
		},
		Commands: commands.Get(app.Config, app.Logger),
	}
	return app
}
