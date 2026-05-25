package guesschema_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Olian04/guesschema/pkg/guesschema"
)

func TestGuesser_Run_EOF(t *testing.T) {
	t.Parallel()
	in := strings.NewReader(`{"a":1}` + "\n")
	var out bytes.Buffer
	ctx := context.Background()
	g, err := guesschema.New(
		guesschema.WithReadWindow(time.Second),
		guesschema.WithVariantThreshold(0.1),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Run(ctx, in, &out); err != nil {
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

func TestGuesser_Run_noExtra(t *testing.T) {
	t.Parallel()
	in := strings.NewReader(`{"a":1}` + "\n")
	var out bytes.Buffer
	ctx := context.Background()
	g, err := guesschema.New(
		guesschema.WithReadWindow(time.Second),
		guesschema.WithVariantThreshold(0.1),
		guesschema.WithOmitVendorExtensions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Run(ctx, in, &out); err != nil {
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
	props, ok := doc["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties map, got %T", doc["properties"])
	}
	a, ok := props["a"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties.a map, got %T", props["a"])
	}
	for k := range a {
		if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
			t.Fatalf("unexpected nested vendor key in properties.a: %q", k)
		}
	}
}

func TestNew_invalidReadWindow(t *testing.T) {
	t.Parallel()
	_, err := guesschema.New(guesschema.WithReadWindow(0))
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestGuesser_runConcurrent(t *testing.T) {
	t.Parallel()
	g, err := guesschema.New(
		guesschema.WithReadWindow(100*time.Millisecond),
		guesschema.WithVariantThreshold(0.1),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var out bytes.Buffer
			in := strings.NewReader(`{"n":1}` + "\n")
			if err := g.Run(ctx, in, &out); err != nil {
				t.Error(err)
				return
			}
			var doc map[string]any
			if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}
