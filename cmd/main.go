package main

import (
	"context"
	"os"

	"github.com/got-many-wheels/spoderman/internal/app"
)

func main() {
	app := app.New()
	if err := app.Cli.Run(context.Background(), os.Args); err != nil {
		panic(err)
	}
}
