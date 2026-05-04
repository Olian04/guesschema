package schemaguess_test

import (
	"encoding/json"
	"slices"
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
	var names []string
	switch v := s["required"].(type) {
	case []string:
		names = v
	case []any:
		for _, r := range v {
			names = append(names, r.(string))
		}
	default:
		t.Fatalf("unexpected required type %T", s["required"])
	}
	if !slices.Contains(names, "x") {
		t.Fatalf("x present on every line → required; got required %#v", names)
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
	// Key x on every line with varying types → combined coverage == lines_total → still required.
	var req []string
	switch v := s["required"].(type) {
	case []string:
		req = v
	case []any:
		for _, r := range v {
			req = append(req, r.(string))
		}
	default:
		t.Fatalf("unexpected required type %T", s["required"])
	}
	if !slices.Contains(req, "x") {
		t.Fatalf("oneOf branches sum to full line coverage → x required; got %#v", req)
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

func TestBuildSchema_requiredOmittedWhenNotOnAllLines(t *testing.T) {
	t.Parallel()
	a := schemaguess.NewAccumulator()
	_ = a.ObserveLine([]byte(`{"a":1}`))
	_ = a.ObserveLine([]byte(`{"a":2}`))
	_ = a.ObserveLine([]byte(`{}`))
	s := schemaguess.BuildSchema(a, 0.1, time.Unix(4, 0).UTC())
	props := s["properties"].(map[string]any)
	if _, ok := props["a"]; !ok {
		t.Fatal("expected property a")
	}
	if _, has := s["required"]; has {
		t.Fatalf("a missing on one line → not all likelihood 1; should not set required, got %#v", s["required"])
	}
}
