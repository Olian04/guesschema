package guesschema

import (
	"errors"
	"log/slog"
	"time"
)

const (
	defaultReadWindow       = time.Second
	defaultVariantThreshold = 0.1
)

// Option configures New.
type Option func(*config)

type config struct {
	readWindow       time.Duration
	variantThreshold float64
	omitVendorExt    bool
	startOnNextMsg   bool
	logger           *slog.Logger
}

func defaultConfig() config {
	return config{
		readWindow:       defaultReadWindow,
		variantThreshold: defaultVariantThreshold,
		logger:           slog.New(slog.DiscardHandler),
	}
}

// WithReadWindow sets the maximum wall-clock time to read JSONL per Run window.
func WithReadWindow(d time.Duration) Option {
	return func(c *config) {
		c.readWindow = d
	}
}

// WithVariantThreshold sets T for same-path oneOf vs single winner (0 < T < 1).
func WithVariantThreshold(t float64) Option {
	return func(c *config) {
		c.variantThreshold = t
	}
}

// WithOmitVendorExtensions strips object keys starting with "x-" before writing output.
func WithOmitVendorExtensions() Option {
	return func(c *config) {
		c.omitVendorExt = true
	}
}

// WithStartWindowOnNextMessage starts each read window only after the first JSONL line.
func WithStartWindowOnNextMessage() Option {
	return func(c *config) {
		c.startOnNextMsg = true
	}
}

// WithLogger sets the logger used for diagnostics during Run. Nil keeps the discard logger.
func WithLogger(l *slog.Logger) Option {
	return func(c *config) {
		if l != nil {
			c.logger = l
		}
	}
}

func (c *config) validate() error {
	if c.readWindow < time.Millisecond {
		return errors.New("read-window must be at least 1ms")
	}
	if c.variantThreshold <= 0 || c.variantThreshold >= 1 {
		return errors.New("variant-threshold must satisfy 0 < T < 1")
	}
	return nil
}
