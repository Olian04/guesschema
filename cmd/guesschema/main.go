package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/Olian04/guesschema/cmd/guesschema/version"
	"github.com/Olian04/guesschema/internal/app"
)

func main() {
	vi := version.Info()

	cli.VersionPrinter = func(cmd *cli.Command) {
		_, err := fmt.Fprintf(cmd.Root().Writer, "%s version %s\nrevision %s\nbuild_time %s\n",
			cmd.Name, vi.Version, vi.Revision, vi.BuildTime)
		if err != nil {
			slog.Error("write version", "error", err.Error())
		}
	}

	var (
		readWindow       time.Duration
		every            time.Duration
		variantThreshold float64
	)
	root := &cli.Command{
		Name:    "guesschema",
		Usage:   "Read JSON Lines from stdin, write guessed JSON Schema (2020-12) to stdout",
		Version: vi.Version,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "once",
				Usage: "Single emit after read window (or EOF); default when --every is not set",
			},
			&cli.DurationFlag{
				Name:        "every",
				Usage:       "Emit repeatedly with this period (ticker + spacing between window starts); mutually exclusive with --once",
				Destination: &every,
			},
			&cli.DurationFlag{
				Name:        "read-window",
				Usage:       "Max wall time to read stdin per window (default 1s if omitted)",
				Destination: &readWindow,
			},
			&cli.FloatFlag{
				Name:        "variant-threshold",
				Usage:       "Threshold T for same-path oneOf vs single winner (default 0.1)",
				Value:       0.1,
				Destination: &variantThreshold,
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "Log to stderr (long form only; no short alias)",
			},
			&cli.BoolFlag{
				Name:  "no-extra",
				Usage: "Omit JSON Schema vendor extensions (strip object keys starting with x- from output)",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			debug := c.Bool("debug")
			setupLogging(debug)

			if c.IsSet("every") && every <= 0 {
				return fmt.Errorf("--every must be positive")
			}
			periodic := c.IsSet("every") && every > 0
			if periodic && c.Bool("once") {
				return fmt.Errorf("cannot use --once with --every")
			}
			if err := app.ValidateGuesschemaFlags(readWindow, every, variantThreshold, c.Bool("once"), periodic); err != nil {
				return err
			}

			cfg := app.GuesschemaConfig{
				ReadWindow:       readWindow,
				Every:            every,
				VariantThreshold: variantThreshold,
				NoExtra:          c.Bool("no-extra"),
				Debug:            debug,
			}
			return app.RunGuesschema(ctx, os.Stdin, os.Stdout, cfg)
		},
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := root.Run(ctx, os.Args); err != nil {
		slog.Error("guesschema exited with error", "error", err)
		os.Exit(1)
	}
}

func setupLogging(debug bool) {
	var h slog.Handler
	if debug {
		h = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true})
	} else {
		h = slog.DiscardHandler
	}
	slog.SetDefault(slog.New(h))
}
