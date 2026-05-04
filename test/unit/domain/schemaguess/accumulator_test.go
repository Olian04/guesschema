package schemaguess_test

import (
	"testing"

	"github.com/Olian04/guesschema/internal/domain/schemaguess"
)

func TestObserveLine_missingKeyRules(t *testing.T) {
	t.Parallel()
	a := schemaguess.NewAccumulator()
	// Line 0: object with only "a"
	if err := a.ObserveLine([]byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	// Line 1: object with "a" and "b" — "b" first sight → undefined backfill +1 for linesCompletedBeforeCurrent=1
	if err := a.ObserveLine([]byte(`{"a":2,"b":true}`)); err != nil {
		t.Fatal(err)
	}
	u := a.Variants[schemaguess.VariantKey{Path: "/b", Type: schemaguess.TypeUndefined, Hint: ""}]
	if u == nil || u.LinesWith != 1 {
		t.Fatalf("first-sight /b undefined want 1 got %#v", u)
	}
	// Line 2: missing known key "b"
	if err := a.ObserveLine([]byte(`{"a":3}`)); err != nil {
		t.Fatal(err)
	}
	u = a.Variants[schemaguess.VariantKey{Path: "/b", Type: schemaguess.TypeUndefined, Hint: ""}]
	if u.LinesWith != 2 {
		t.Fatalf("after absence /b undefined want 2 got %d", u.LinesWith)
	}
}
