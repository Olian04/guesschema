package integration

import (
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Olian04/guesschema/test/blackbox"
)

func TestExecutableJSONL(t *testing.T) {
	t.Parallel()
	jsonl := strings.Join([]string{
		`{"a":1,"b":"x"}`,
		`{"a":2}`,
		"",
	}, "\n")
	bin := blackbox.BuildBinary(t)
	res := blackbox.RunBinary(bin, []string{"--read-window", "3s"}, jsonl)
	if res.Err != nil {
		t.Fatalf("run binary: %v\nstderr:\n%s\nstdout:\n%s", res.Err, res.Stderr, res.Stdout)
	}
	doc := blackbox.ParseSingleJSONLine(t, res.Stdout, res.Stderr)

	if doc["$schema"] == nil {
		t.Fatalf("missing $schema\nstdout:\n%s\nstderr:\n%s", res.Stdout, res.Stderr)
	}
	props, ok := doc["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties missing or wrong type %T\nstdout:\n%s\nstderr:\n%s", doc["properties"], res.Stdout, res.Stderr)
	}
	if _, ok := props["a"]; !ok {
		t.Fatalf("missing property a\nstdout:\n%s\nstderr:\n%s", res.Stdout, res.Stderr)
	}
	if _, ok := props["b"]; !ok {
		t.Fatalf("missing property b\nstdout:\n%s\nstderr:\n%s", res.Stdout, res.Stderr)
	}

	req := blackbox.RequiredNames(t, doc, res.Stderr)
	if !slices.Contains(req, "a") {
		t.Fatalf("a should be required\nrequired=%v\nstdout:\n%s\nstderr:\n%s", req, res.Stdout, res.Stderr)
	}
	if slices.Contains(req, "b") {
		t.Fatalf("b should be optional\nrequired=%v\nstdout:\n%s\nstderr:\n%s", req, res.Stdout, res.Stderr)
	}
}

func TestExecutableNoExtra(t *testing.T) {
	t.Parallel()
	bin := blackbox.BuildBinary(t)
	res := blackbox.RunBinary(bin, []string{"--read-window", "2s", "--no-extra"}, "{\"a\":1}\n")
	if res.Err != nil {
		t.Fatalf("run binary: %v\nstderr:\n%s\nstdout:\n%s", res.Err, res.Stderr, res.Stdout)
	}
	doc := blackbox.ParseSingleJSONLine(t, res.Stdout, res.Stderr)
	for k := range doc {
		if strings.HasPrefix(k, "x-") {
			t.Fatalf("unexpected x-* key at root: %q\nstdout:\n%s\nstderr:\n%s", k, res.Stdout, res.Stderr)
		}
	}
}

func TestExecutableWithNoInput(t *testing.T) {
	t.Parallel()
	bin := blackbox.BuildBinary(t)
	res := blackbox.RunBinary(bin, []string{"--read-window", "1s"}, "")
	if res.Err != nil {
		t.Fatalf("run binary: %v\nstderr:\n%s\nstdout:\n%s", res.Err, res.Stderr, res.Stdout)
	}
	doc := blackbox.ParseSingleJSONLine(t, res.Stdout, res.Stderr)
	if doc["$schema"] == nil {
		t.Fatalf("missing $schema for empty input run\nstdout:\n%s\nstderr:\n%s", res.Stdout, res.Stderr)
	}
}

func TestExecutableStartWindowOnNextMessage(t *testing.T) {
	t.Parallel()
	bin := blackbox.BuildBinary(t)
	res := blackbox.RunBinaryStreaming(
		bin,
		[]string{"--read-window", "500ms", "--start-window-on-next-message"},
		func(w io.WriteCloser) {
			time.Sleep(800 * time.Millisecond)
			_, _ = io.WriteString(w, `{"late":1}`+"\n")
		},
	)
	if res.Err != nil {
		t.Fatalf("run binary: %v\nstderr:\n%s\nstdout:\n%s", res.Err, res.Stderr, res.Stdout)
	}
	doc := blackbox.ParseSingleJSONLine(t, res.Stdout, res.Stderr)
	props, ok := doc["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties map\nstdout:\n%s\nstderr:\n%s", res.Stdout, res.Stderr)
	}
	if _, ok := props["late"]; !ok {
		t.Fatalf("expected late key from delayed input\nstdout:\n%s\nstderr:\n%s", res.Stdout, res.Stderr)
	}
}
