package integration

import (
	"strings"
	"testing"

	"github.com/Olian04/guesschema/test/blackbox"
)

func TestExecutableInvalidFlags(t *testing.T) {
	t.Parallel()
	bin := blackbox.BuildBinary(t)
	cases := []struct {
		name string
		args []string
	}{
		{name: "negative read-window", args: []string{"--read-window", "-1s"}},
		{name: "read-window below 1ms", args: []string{"--read-window", "500us"}},
		{name: "variant threshold zero", args: []string{"--variant-threshold", "0"}},
		{name: "variant threshold one", args: []string{"--variant-threshold", "1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := blackbox.RunBinary(bin, tc.args, "")
			if res.Err == nil {
				t.Fatalf("expected non-zero exit\nstdout:\n%s\nstderr:\n%s", res.Stdout, res.Stderr)
			}
		})
	}
}

func TestExecutableDefaultReadWindow(t *testing.T) {
	t.Parallel()
	bin := blackbox.BuildBinary(t)
	res := blackbox.RunBinary(bin, nil, "{\"a\":1}\n")
	if res.Err != nil {
		t.Fatalf("run binary: %v\nstderr:\n%s\nstdout:\n%s", res.Err, res.Stderr, res.Stdout)
	}
	if strings.TrimSpace(res.Stdout) == "" {
		t.Fatal("expected schema on stdout")
	}
}
