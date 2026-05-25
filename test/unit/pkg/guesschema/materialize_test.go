package guesschema_test

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/Olian04/guesschema/pkg/guesschema"
)

func TestBuildSchema_strategyB(t *testing.T) {
	t.Parallel()
	a := guesschema.NewAccumulator()
	_ = a.ObserveLine([]byte(`{"x":1}`))
	_ = a.ObserveLine([]byte(`{"x":1}`))
	_ = a.ObserveLine([]byte(`{"x":"hi"}`))
	s := guesschema.BuildSchema(a, 0.5, time.Unix(1, 0).UTC())
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
	a := guesschema.NewAccumulator()
	for range 4 {
		_ = a.ObserveLine([]byte(`{"x":1}`))
	}
	for range 3 {
		_ = a.ObserveLine([]byte(`{"x":"a"}`))
	}
	for range 3 {
		_ = a.ObserveLine([]byte(`{"x":true}`))
	}
	s := guesschema.BuildSchema(a, 0.1, time.Unix(2, 0).UTC())
	props := s["properties"].(map[string]any)
	x := props["x"].(map[string]any)
	if _, ok := x["oneOf"]; !ok {
		out, _ := json.MarshalIndent(x, "", "  ")
		t.Fatalf("expected oneOf at x, got %s", out)
	}
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
	a := guesschema.NewAccumulator()
	for range 9 {
		_ = a.ObserveLine([]byte(`{"x":1}`))
	}
	_ = a.ObserveLine([]byte(`{"x":"rare"}`))
	s := guesschema.BuildSchema(a, 0.1, time.Unix(3, 0).UTC())
	props := s["properties"].(map[string]any)
	x := props["x"].(map[string]any)
	if x["type"] != "number" {
		t.Fatalf("want single winner number, got %#v", x)
	}
}

func TestBuildSchema_requiredOmittedWhenNotOnAllLines(t *testing.T) {
	t.Parallel()
	a := guesschema.NewAccumulator()
	_ = a.ObserveLine([]byte(`{"a":1}`))
	_ = a.ObserveLine([]byte(`{"a":2}`))
	_ = a.ObserveLine([]byte(`{}`))
	s := guesschema.BuildSchema(a, 0.1, time.Unix(4, 0).UTC())
	props := s["properties"].(map[string]any)
	if _, ok := props["a"]; !ok {
		t.Fatal("expected property a")
	}
	if _, has := s["required"]; has {
		t.Fatalf("a missing on one line → not all likelihood 1; should not set required, got %#v", s["required"])
	}
}

func TestBuildSchema_optionalKeyAfterAbsence(t *testing.T) {
	t.Parallel()
	a := guesschema.NewAccumulator()
	_ = a.ObserveLine([]byte(`{"a":1}`))
	_ = a.ObserveLine([]byte(`{"a":2,"b":true}`))
	_ = a.ObserveLine([]byte(`{"a":3}`))
	s := guesschema.BuildSchema(a, 0.1, time.Unix(5, 0).UTC())
	props := s["properties"].(map[string]any)
	if _, ok := props["b"]; !ok {
		t.Fatal("expected optional property b")
	}
	if req, ok := s["required"].([]any); ok {
		for _, r := range req {
			if r == "b" {
				t.Fatalf("b only on one line → not required, got %#v", s["required"])
			}
		}
	}
	if req, ok := s["required"].([]string); ok && slices.Contains(req, "b") {
		t.Fatalf("b only on one line → not required, got %#v", req)
	}
}

func TestBuildSchema_escapedPropertyName(t *testing.T) {
	t.Parallel()
	a := guesschema.NewAccumulator()
	if err := a.ObserveLine([]byte(`{"b/c":1}`)); err != nil {
		t.Fatal(err)
	}
	s := guesschema.BuildSchema(a, 0.1, time.Unix(6, 0).UTC())
	props := s["properties"].(map[string]any)
	if _, ok := props["b/c"]; !ok {
		t.Fatalf("expected property b/c, got keys %#v", props)
	}
}
