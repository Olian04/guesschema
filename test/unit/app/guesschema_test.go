package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Olian04/guesschema/internal/app"
)

func TestRunGuesschema_once_EOF(t *testing.T) {
	t.Parallel()
	in := strings.NewReader(`{"a":1}` + "\n")
	var out bytes.Buffer
	ctx := context.Background()
	cfg := app.GuesschemaConfig{ReadWindow: time.Second, VariantThreshold: 0.1}
	if err := app.RunGuesschema(ctx, in, &out, cfg); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["type"] != "object" {
		t.Fatalf("root type: %#v", doc["type"])
	}
	if _, ok := doc["x-guesschema-generated-at"]; !ok {
		t.Fatal("expected default output to include x-guesschema-generated-at")
	}
}

func TestRunGuesschema_noExtra(t *testing.T) {
	t.Parallel()
	in := strings.NewReader(`{"a":1}` + "\n")
	var out bytes.Buffer
	ctx := context.Background()
	cfg := app.GuesschemaConfig{ReadWindow: time.Second, VariantThreshold: 0.1, NoExtra: true}
	if err := app.RunGuesschema(ctx, in, &out, cfg); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	for k := range doc {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
			t.Fatalf("unexpected vendor key in output: %q", k)
		}
	}
}

func TestValidateGuesschemaFlags(t *testing.T) {
	t.Parallel()
	if err := app.ValidateGuesschemaFlags(2*time.Second, time.Second, 0.1, false, true); err == nil {
		t.Fatal("expected error read-window > every")
	}
	if err := app.ValidateGuesschemaFlags(0, 500*time.Millisecond, 0.1, false, true); err == nil {
		t.Fatal("expected error every < 1s default budget")
	}
}
