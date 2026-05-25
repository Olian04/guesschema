package guesschema

import (
	"log/slog"
	"time"

	ig "github.com/Olian04/guesschema/internal/guesschema"
)

// WithReadWindow sets the maximum wall-clock time to read JSONL per Run window.
//
// If this option is not passed, New configures a 1 second read window. The window
// ends on timeout, context cancellation, or EOF (whichever comes first).
//
// Example (wait up to 5s for lines before emitting one schema):
//
//	g, err := guesschema.New(guesschema.WithReadWindow(5 * time.Second))
//	if err != nil {
//	    return err
//	}
//	err = g.Run(ctx, in, out)
//
// Example (tight 100ms window for fast tests):
//
//	g, err := guesschema.New(guesschema.WithReadWindow(100 * time.Millisecond))
//	if err != nil {
//	    return err
//	}
//	err = g.Run(ctx, in, out)
func WithReadWindow(d time.Duration) Option { return ig.WithReadWindow(d) }

// WithVariantThreshold sets T for same-path oneOf vs single winner at each path.
// Valid range is 0 < T < 1. Default is 0.1 when this option is omitted.
//
// Example (stricter oneOf: only when every variant has likelihood > 0.5):
//
//	g, err := guesschema.New(guesschema.WithVariantThreshold(0.5))
//	if err != nil {
//	    return err
//	}
//	err = g.Run(ctx, in, out)
//
// Example (looser: prefer single winning type unless variants are very balanced):
//
//	g, err := guesschema.New(guesschema.WithVariantThreshold(0.05))
//	if err != nil {
//	    return err
//	}
//	err = g.Run(ctx, in, out)
func WithVariantThreshold(t float64) Option { return ig.WithVariantThreshold(t) }

// WithOmitVendorExtensions removes all JSON object keys starting with "x-" from
// output (including guesschema metadata keys such as x-guesschema-generated-at).
//
// Example (consumer-facing schema without vendor extensions):
//
//	g, err := guesschema.New(guesschema.WithOmitVendorExtensions())
//	if err != nil {
//	    return err
//	}
//	err = g.Run(ctx, in, out)
func WithOmitVendorExtensions() Option { return ig.WithOmitVendorExtensions() }

// WithStartWindowOnNextMessage delays the read-window timer until the first JSONL
// line arrives. Useful when stdin may be idle and you want to avoid an empty-window emit.
//
// Example (block on stdin until first message, then read up to read-window):
//
//	g, err := guesschema.New(guesschema.WithStartWindowOnNextMessage())
//	if err != nil {
//	    return err
//	}
//	err = g.Run(ctx, in, out)
func WithStartWindowOnNextMessage() Option { return ig.WithStartWindowOnNextMessage() }

// WithLogger sets the logger used for Run diagnostics (Debug-level messages).
// If this option is not passed, logs are discarded. Passing nil keeps the discard logger.
//
// Example (debug logs to stderr):
//
//	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
//	g, err := guesschema.New(guesschema.WithLogger(log))
//	if err != nil {
//	    return err
//	}
//	err = g.Run(ctx, in, out)
func WithLogger(l *slog.Logger) Option { return ig.WithLogger(l) }
