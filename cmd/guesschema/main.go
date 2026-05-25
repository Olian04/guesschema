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
	ig "github.com/Olian04/guesschema/internal/guesschema"
)

func main() {
	vi := version.Info()
	log := setupLogging(false)

	cli.VersionPrinter = func(cmd *cli.Command) {
		_, err := fmt.Fprintf(cmd.Root().Writer, "%s version %s\nrevision %s\nbuild_time %s\n",
			cmd.Name, vi.Version, vi.Revision, vi.BuildTime)
		if err != nil {
			log.Error("write version", "error", err.Error())
		}
	}

	var (
		readWindow       time.Duration
		variantThreshold float64
	)
	root := &cli.Command{
		Name:    "guesschema",
		Usage:   "Read JSON Lines from stdin, write guessed JSON Schema (2020-12) to stdout",
		Version: vi.Version,
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name:        "read-window",
				Usage:       "Max wall time to read stdin per window (default 1s)",
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
			&cli.BoolFlag{
				Name:  "start-window-on-next-message",
				Usage: "Start read-window only after first received JSONL line",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			log = setupLogging(c.Bool("debug"))

			if err := validateFlags(readWindow, variantThreshold); err != nil {
				return err
			}

			opts := []ig.Option{ig.WithLogger(log)}
			if readWindow > 0 {
				opts = append(opts, ig.WithReadWindow(readWindow))
			}
			if c.IsSet("variant-threshold") {
				opts = append(opts, ig.WithVariantThreshold(variantThreshold))
			}
			if c.Bool("no-extra") {
				opts = append(opts, ig.WithOmitVendorExtensions())
			}
			if c.Bool("start-window-on-next-message") {
				opts = append(opts, ig.WithStartWindowOnNextMessage())
			}

			g, err := ig.New(opts...)
			if err != nil {
				return err
			}
			return g.Run(ctx, os.Stdin, os.Stdout)
		},
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := root.Run(ctx, os.Args); err != nil {
		log.Error("guesschema exited with error", "error", err)
		os.Exit(1)
	}
}

func setupLogging(debug bool) *slog.Logger {
	var h slog.Handler
	if debug {
		h = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true})
	} else {
		h = slog.DiscardHandler
	}
	log := slog.New(h)
	slog.SetDefault(log)
	return log
}
