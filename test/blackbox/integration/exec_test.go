package integration

import (
	"encoding/json"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Olian04/guesschema/test/blackbox"
)

func TestExecutableOnceJSONL(t *testing.T) {
	t.Parallel()
	jsonl := strings.Join([]string{
		`{"a":1,"b":"x"}`,
		`{"a":2}`,
		"",
	}, "\n")
	bin := blackbox.BuildBinary(t)
	res := blackbox.RunBinary(bin, []string{"--once", "--read-window", "3s"}, jsonl)
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
	res := blackbox.RunBinary(bin, []string{"--once", "--read-window", "2s", "--no-extra"}, "{\"a\":1}\n")
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

func TestExecutableEveryWithFiniteInput(t *testing.T) {
	t.Parallel()
	jsonl := strings.Join([]string{
		`{"x":1}`,
		`{"x":"two"}`,
		"",
	}, "\n")
	bin := blackbox.BuildBinary(t)
	res := blackbox.RunBinary(bin, []string{"--every", "2s", "--read-window", "1s"}, jsonl)
	if res.Err != nil {
		t.Fatalf("run binary: %v\nstderr:\n%s\nstdout:\n%s", res.Err, res.Stderr, res.Stdout)
	}
	lines := strings.Split(strings.TrimSpace(res.Stdout), "\n")
	if len(lines) < 1 {
		t.Fatalf("expected at least 1 schema line\nstdout:\n%s\nstderr:\n%s", res.Stdout, res.Stderr)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &doc); err != nil {
		t.Fatalf("invalid first JSON line: %v\nstdout:\n%s\nstderr:\n%s", err, res.Stdout, res.Stderr)
	}
	if doc["$schema"] == nil {
		t.Fatalf("missing $schema in periodic output\nstdout:\n%s\nstderr:\n%s", res.Stdout, res.Stderr)
	}
}

func TestExecutableEveryWithNonFiniteInput(t *testing.T) {
	t.Parallel()
	bin := blackbox.BuildBinary(t)
	res := blackbox.RunBinaryStreaming(bin, []string{"--every", "1s", "--read-window", "200ms"}, func(w io.WriteCloser) {
		_, _ = io.WriteString(w, `{"a":1}`+"\n")
		time.Sleep(1300 * time.Millisecond)
		_, _ = io.WriteString(w, `{"b":"late"}`+"\n")
		time.Sleep(100 * time.Millisecond)
	})
	if res.Err != nil {
		t.Fatalf("run binary: %v\nstderr:\n%s\nstdout:\n%s", res.Err, res.Stderr, res.Stdout)
	}
	lines := strings.Split(strings.TrimSpace(res.Stdout), "\n")
	if len(lines) < 1 {
		t.Fatalf("expected at least 1 schema line for streaming input\nstdout:\n%s\nstderr:\n%s", res.Stdout, res.Stderr)
	}
	seenA := false
	seenB := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var doc map[string]any
		if err := json.Unmarshal([]byte(line), &doc); err != nil {
			t.Fatalf("invalid json line: %v\nline:\n%s\nstdout:\n%s\nstderr:\n%s", err, line, res.Stdout, res.Stderr)
		}
		props, _ := doc["properties"].(map[string]any)
		if _, ok := props["a"]; ok {
			seenA = true
		}
		if _, ok := props["b"]; ok {
			seenB = true
		}
	}
	if !seenA || !seenB {
		t.Fatalf("expected streamed outputs to include both early and late keys (seenA=%v seenB=%v)\nstdout:\n%s\nstderr:\n%s", seenA, seenB, res.Stdout, res.Stderr)
	}
}

func TestExecutableOnceWithNoInput(t *testing.T) {
	t.Parallel()
	bin := blackbox.BuildBinary(t)
	res := blackbox.RunBinary(bin, []string{"--once", "--read-window", "1s"}, "")
	if res.Err != nil {
		t.Fatalf("run binary: %v\nstderr:\n%s\nstdout:\n%s", res.Err, res.Stderr, res.Stdout)
	}
	doc := blackbox.ParseSingleJSONLine(t, res.Stdout, res.Stderr)
	if doc["$schema"] == nil {
		t.Fatalf("missing $schema for empty input run\nstdout:\n%s\nstderr:\n%s", res.Stdout, res.Stderr)
	}
}

func TestExecutableOnceStartWindowOnNextMessage(t *testing.T) {
	t.Parallel()
	bin := blackbox.BuildBinary(t)
	res := blackbox.RunBinaryStreaming(
		bin,
		[]string{"--once", "--read-window", "500ms", "--start-window-on-next-message"},
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

func TestExecutableEveryStartWindowOnNextMessage(t *testing.T) {
	t.Parallel()
	bin := blackbox.BuildBinary(t)
	res := blackbox.RunBinaryStreaming(
		bin,
		[]string{"--every", "1s", "--read-window", "300ms", "--start-window-on-next-message"},
		func(w io.WriteCloser) {
			// Wait longer than read-window before first row; with the flag, no empty-window
			// emit should happen and first window should start on this message.
			time.Sleep(900 * time.Millisecond)
			_, _ = io.WriteString(w, `{"first":"arrived"}`+"\n")
			// Keep stream open long enough for first emit, then send another row so we get
			// a subsequent window as well.
			time.Sleep(1200 * time.Millisecond)
			_, _ = io.WriteString(w, `{"second":2}`+"\n")
			time.Sleep(200 * time.Millisecond)
		},
	)
	if res.Err != nil {
		t.Fatalf("run binary: %v\nstderr:\n%s\nstdout:\n%s", res.Err, res.Stderr, res.Stdout)
	}
	lines := strings.Split(strings.TrimSpace(res.Stdout), "\n")
	if len(lines) < 1 {
		t.Fatalf("expected at least 1 schema line\nstdout:\n%s\nstderr:\n%s", res.Stdout, res.Stderr)
	}
	seenFirst := false
	seenSecond := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var doc map[string]any
		if err := json.Unmarshal([]byte(line), &doc); err != nil {
			t.Fatalf("invalid json line: %v\nline:\n%s\nstdout:\n%s\nstderr:\n%s", err, line, res.Stdout, res.Stderr)
		}
		props, _ := doc["properties"].(map[string]any)
		if _, ok := props["first"]; ok {
			seenFirst = true
		}
		if _, ok := props["second"]; ok {
			seenSecond = true
		}
	}
	if !seenFirst || !seenSecond {
		t.Fatalf("expected outputs to include both delayed-first and later-second keys (seenFirst=%v seenSecond=%v)\nstdout:\n%s\nstderr:\n%s", seenFirst, seenSecond, res.Stdout, res.Stderr)
	}
}
