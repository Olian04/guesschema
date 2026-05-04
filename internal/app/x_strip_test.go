package app

import (
	"testing"
)

func TestStripXVendorKeys(t *testing.T) {
	t.Parallel()
	doc := map[string]any{
		"$schema":                         "https://json-schema.org/draft/2020-12/schema",
		"x-guesschema-generated-at":       "2026-01-01T00:00:00Z",
		"x-guesschema-invalid-json-lines": float64(1),
		"type":                            "object",
		"properties": map[string]any{
			"a": map[string]any{
				"type":                       "string",
				"x-guesschema-lines-with":    float64(2),
				"x-guesschema-lines-total":   float64(3),
				"x-guesschema-likelihood":    0.5,
				"x-guesschema-custom-ignore": "bye",
			},
		},
	}
	stripXVendorKeys(doc)
	if _, ok := doc["x-guesschema-generated-at"]; ok {
		t.Fatal("expected root x-* removed")
	}
	a := doc["properties"].(map[string]any)["a"].(map[string]any)
	if _, ok := a["x-guesschema-lines-with"]; ok {
		t.Fatal("expected nested x-* removed")
	}
	if doc["$schema"] == nil {
		t.Fatal("$schema must remain")
	}
	if a["type"] != "string" {
		t.Fatal("non-x keys must remain")
	}
}
