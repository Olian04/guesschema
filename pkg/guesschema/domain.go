package guesschema

import (
	"time"

	ig "github.com/Olian04/guesschema/internal/guesschema"
)

// Accumulator holds cross-line schema guess state for one read window.
type Accumulator = ig.Accumulator

// NewAccumulator returns an empty accumulator.
func NewAccumulator() *Accumulator { return ig.NewAccumulator() }

// BuildSchema returns a JSON Schema 2020-12 root document map.
func BuildSchema(acc *Accumulator, variantThreshold float64, generatedAt time.Time) map[string]any {
	return ig.BuildSchema(acc, variantThreshold, generatedAt)
}
