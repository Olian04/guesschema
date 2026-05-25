package regression

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/Olian04/guesschema/test/blackbox"
)

const (
	regressionCases = 12
	maxLineBytes    = 16 << 20
)

type caseMode int

const (
	modeValidOnly caseMode = iota
	modeNonFatalInvalidRows
	modeFatalOversizeRow
)

func TestRandomExecutableRegression(t *testing.T) {
	t.Parallel()

	seed := time.Now().UnixNano()
	rng := rand.New(rand.NewSource(seed))
	bin := blackbox.BuildBinary(t)

	for i := range regressionCases {
		mode := caseMode(rng.Intn(3))
		input := makeRandomJSONL(rng, mode)
		res := blackbox.RunBinary(bin, []string{"--read-window", "5s"}, input)

		expectSuccess := mode != modeFatalOversizeRow
		if expectSuccess {
			if res.Err != nil {
				t.Fatalf("seed=%d case=%d mode=%v expected success got error=%v\nstderr:\n%s\nstdout:\n%s",
					seed, i, mode, res.Err, res.Stderr, res.Stdout)
			}
			doc := blackbox.ParseSingleJSONLine(t, res.Stdout, res.Stderr)
			if doc["$schema"] == nil {
				t.Fatalf("seed=%d case=%d mode=%v missing $schema\nstderr:\n%s\nstdout:\n%s",
					seed, i, mode, res.Stderr, res.Stdout)
			}
			continue
		}

		if res.Err == nil {
			t.Fatalf("seed=%d case=%d mode=%v expected failure (oversized line) but got success\nstderr:\n%s\nstdout:\n%s",
				seed, i, mode, res.Stderr, res.Stdout)
		}
	}
}

func makeRandomJSONL(rng *rand.Rand, mode caseMode) string {
	rowCount := 1 + rng.Intn(40)
	lines := make([]string, 0, rowCount+2)
	oversizePos := -1
	if mode == modeFatalOversizeRow {
		oversizePos = rng.Intn(rowCount)
	}
	for i := 0; i < rowCount; i++ {
		switch {
		case mode == modeFatalOversizeRow && i == oversizePos:
			lines = append(lines, makeOversizeInvalidLine(rng))
		case mode != modeValidOnly && rng.Intn(5) == 0:
			lines = append(lines, randomInvalidJSONLine(rng))
		default:
			lines = append(lines, randomValidJSONObjectLine(rng))
		}
	}
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func randomValidJSONObjectLine(rng *rand.Rand) string {
	obj := randomObject(rng, 0)
	b, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func randomObject(rng *rand.Rand, depth int) map[string]any {
	n := 1 + rng.Intn(6)
	out := make(map[string]any, n)
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("k%d_%d", depth, i)
		out[key] = randomValue(rng, depth+1)
	}
	return out
}

func randomValue(rng *rand.Rand, depth int) any {
	if depth > 2 {
		return randomLeaf(rng)
	}
	switch rng.Intn(6) {
	case 0:
		return randomLeaf(rng)
	case 1:
		return randomLeaf(rng)
	case 2:
		return randomLeaf(rng)
	case 3:
		return randomObject(rng, depth)
	case 4:
		n := rng.Intn(5)
		arr := make([]any, 0, n)
		for i := 0; i < n; i++ {
			arr = append(arr, randomLeaf(rng))
		}
		return arr
	default:
		return randomLeaf(rng)
	}
}

func randomLeaf(rng *rand.Rand) any {
	switch rng.Intn(5) {
	case 0:
		return nil
	case 1:
		return rng.Intn(2) == 0
	case 2:
		return rng.Intn(1000) - 500
	case 3:
		return rng.Float64() * 100
	default:
		return fmt.Sprintf("s_%d", rng.Intn(100000))
	}
}

func randomInvalidJSONLine(rng *rand.Rand) string {
	choices := []string{
		`{"unterminated":`,
		`{"x": 1,, "y":2}`,
		`not-json`,
		`{"a": [1,2,}`,
	}
	return choices[rng.Intn(len(choices))]
}

func makeOversizeInvalidLine(rng *rand.Rand) string {
	// Slightly above scanner hard limit to force a fatal read error.
	n := maxLineBytes + 1 + rng.Intn(4096)
	return strings.Repeat("x", n)
}
