// Package guesschema reads JSON Lines and writes one guessed JSON Schema (2020-12) per Run.
// Configure a Guesser with New and options, then call Run per stream.
// NewAccumulator and BuildSchema support custom pipelines without time windows.
package guesschema

import (
	ig "github.com/Olian04/guesschema/internal/guesschema"
)

type (
	// Option configures New.
	Option = ig.Option
	// Guesser holds frozen configuration for inferring JSON Schema from JSONL.
	// It has no per-run state and is safe for concurrent Run calls from multiple
	// goroutines. Do not share the same io.Reader across concurrent Run calls.
	Guesser = ig.Guesser
)

// New applies opts, validates, and returns a Guesser safe for concurrent use.
func New(opts ...Option) (*Guesser, error) { return ig.New(opts...) }
