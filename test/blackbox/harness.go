package blackbox

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Result captures process outputs and execution error from running the binary.
type Result struct {
	Stdout string
	Stderr string
	Err    error
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// callers live in test/blackbox/{integration,regression}
	return filepath.Clean(filepath.Join(wd, "..", "..", ".."))
}

// BuildBinary compiles cmd/guesschema and returns a path to temp executable.
func BuildBinary(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	bin := filepath.Join(t.TempDir(), "guessschema")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/guesschema")
	cmd.Dir = root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build binary: %v\nstderr:\n%s", err, stderr.String())
	}
	return bin
}

// RunBinary runs pre-built binary with args and stdin JSONL payload.
func RunBinary(binaryPath string, args []string, jsonl string) Result {
	cmd := exec.Command(binaryPath, args...)
	cmd.Stdin = strings.NewReader(jsonl)
	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	err := cmd.Run()
	return Result{
		Stdout: out.String(),
		Stderr: errBuf.String(),
		Err:    err,
	}
}

// RunBinaryStreaming runs binary and lets caller stream stdin over time.
// writer should write complete JSONL rows (including trailing newlines as needed).
func RunBinaryStreaming(binaryPath string, args []string, writer func(io.WriteCloser)) Result {
	cmd := exec.Command(binaryPath, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return Result{Err: err}
	}
	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf

	if err := cmd.Start(); err != nil {
		return Result{Stderr: errBuf.String(), Err: err}
	}
	go func() {
		writer(stdin)
		_ = stdin.Close()
	}()

	err = cmd.Wait()
	return Result{
		Stdout: out.String(),
		Stderr: errBuf.String(),
		Err:    err,
	}
}

// ParseSingleJSONLine parses exactly one JSON object line from stdout.
func ParseSingleJSONLine(t *testing.T, stdout, stderr string) map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 1 {
		t.Fatalf("want 1 stdout line, got %d\nstdout:\n%s\nstderr:\n%s", len(lines), stdout, stderr)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &doc); err != nil {
		t.Fatalf("unmarshal stdout: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	return doc
}

// RequiredNames extracts required list from root schema map.
func RequiredNames(t *testing.T, root map[string]any, stderr string) []string {
	t.Helper()
	raw, ok := root["required"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []any:
		names := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				t.Fatalf("required item type %T\nstderr:\n%s", item, stderr)
			}
			names = append(names, s)
		}
		return names
	case []string:
		return v
	default:
		t.Fatalf("unexpected required type %T\nstderr:\n%s", raw, stderr)
		return nil
	}
}
