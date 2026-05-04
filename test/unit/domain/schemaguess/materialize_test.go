package schemaguess_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Olian04/guesschema/internal/domain/schemaguess"
)

func TestBuildSchema_strategyB(t *testing.T) {
	t.Parallel()
	a := schemaguess.NewAccumulator()
	_ = a.ObserveLine([]byte(`{"x":1}`))
	_ = a.ObserveLine([]byte(`{"x":1}`))
	_ = a.ObserveLine([]byte(`{"x":"hi"}`))
	// T high enough that not every variant clears the bar → Strategy B (single winner).
	s := schemaguess.BuildSchema(a, 0.5, time.Unix(1, 0).UTC())
	if s["$schema"] == nil {
		t.Fatal("missing $schema")
	}
	props := s["properties"].(map[string]any)
	x := props["x"].(map[string]any)
	if x["type"] != "number" {
		t.Fatalf("winner should be number, got %#v", x["type"])
	}
}

func TestBuildSchema_oneOfWhenAllLikely(t *testing.T) {
	t.Parallel()
	a := schemaguess.NewAccumulator()
	for range 4 {
		_ = a.ObserveLine([]byte(`{"x":1}`))
	}
	for range 3 {
		_ = a.ObserveLine([]byte(`{"x":"a"}`))
	}
	for range 3 {
		_ = a.ObserveLine([]byte(`{"x":true}`))
	}
	s := schemaguess.BuildSchema(a, 0.1, time.Unix(2, 0).UTC())
	props := s["properties"].(map[string]any)
	x := props["x"].(map[string]any)
	if _, ok := x["oneOf"]; !ok {
		out, _ := json.MarshalIndent(x, "", "  ")
		t.Fatalf("expected oneOf at x, got %s", out)
	}
}

func TestBuildSchema_variantBelowThresholdUsesWinner(t *testing.T) {
	t.Parallel()
	a := schemaguess.NewAccumulator()
	for range 9 {
		_ = a.ObserveLine([]byte(`{"x":1}`))
	}
	_ = a.ObserveLine([]byte(`{"x":"rare"}`))
	s := schemaguess.BuildSchema(a, 0.1, time.Unix(3, 0).UTC())
	props := s["properties"].(map[string]any)
	x := props["x"].(map[string]any)
	if x["type"] != "number" {
		t.Fatalf("want single winner number, got %#v", x)
	}
}
