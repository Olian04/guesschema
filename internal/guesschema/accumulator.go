package guesschema

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/Olian04/guesschema/internal/hints"
)

// Accumulator holds cross-line schema guess state for one read window.
type Accumulator struct {
	successfulLines int
	invalidJSON     int
	variants        map[variantKey]*variantStats
	knownKeys       map[string]map[string]struct{}
}

// NewAccumulator returns an empty accumulator.
func NewAccumulator() *Accumulator {
	return &Accumulator{
		variants:  make(map[variantKey]*variantStats),
		knownKeys: make(map[string]map[string]struct{}),
	}
}

func (a *Accumulator) stats(key variantKey) *variantStats {
	s := a.variants[key]
	if s == nil {
		s = &variantStats{}
		a.variants[key] = s
	}
	return s
}

// Reset clears all state for a new window.
func (a *Accumulator) Reset() {
	a.successfulLines = 0
	a.invalidJSON = 0
	clear(a.variants)
	clear(a.knownKeys)
}

// ObserveLine parses one JSONL line and updates the accumulator. Empty lines are skipped.
func (a *Accumulator) ObserveLine(line []byte) error {
	trim := trimLine(line)
	if len(trim) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(trim, &v); err != nil {
		a.invalidJSON++
		return nil
	}
	linesBefore := a.successfulLines
	if err := a.walkValue("", "", v, linesBefore); err != nil {
		return err
	}
	a.successfulLines++
	return nil
}

func trimLine(b []byte) []byte {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\t') {
		b = b[1:]
	}
	for len(b) > 0 && (b[len(b)-1] == ' ' || b[len(b)-1] == '\t' || b[len(b)-1] == '\r' || b[len(b)-1] == '\n') {
		b = b[:len(b)-1]
	}
	return b
}

func (a *Accumulator) ensureKnownKeys(p string) map[string]struct{} {
	m := a.knownKeys[p]
	if m == nil {
		m = make(map[string]struct{})
		a.knownKeys[p] = m
	}
	return m
}

func (a *Accumulator) walkValue(ptr, propKey string, v any, linesCompletedBeforeCurrent int) error {
	h := hints.For(propKey, v)
	switch val := v.(type) {
	case nil:
		a.stats(variantKey{Path: ptr, Type: typeNull, Hint: h}).LinesWith++
		return nil
	case bool:
		a.stats(variantKey{Path: ptr, Type: typeBoolean, Hint: h}).LinesWith++
		return nil
	case float64:
		if math.IsInf(val, 0) || math.IsNaN(val) {
			return fmt.Errorf("non-finite number at %q", ptr)
		}
		a.stats(variantKey{Path: ptr, Type: typeNumber, Hint: h}).LinesWith++
		return nil
	case string:
		a.stats(variantKey{Path: ptr, Type: typeString, Hint: h}).LinesWith++
		return nil
	case []any:
		a.stats(variantKey{Path: ptr, Type: typeArray, Hint: h}).LinesWith++
		for i, el := range val {
			child := joinPointer(ptr, fmt.Sprintf("%d", i))
			if err := a.walkValue(child, fmt.Sprintf("%d", i), el, linesCompletedBeforeCurrent); err != nil {
				return err
			}
		}
		return nil
	case map[string]any:
		a.stats(variantKey{Path: ptr, Type: typeObject, Hint: h}).LinesWith++
		known := a.ensureKnownKeys(ptr)
		present := make(map[string]struct{}, len(val))
		for k := range val {
			present[k] = struct{}{}
		}
		for k := range known {
			if _, ok := present[k]; !ok {
				p := joinPointer(ptr, k)
				a.stats(variantKey{Path: p, Type: typeUndefined, Hint: ""}).LinesWith++
			}
		}
		for k, childVal := range val {
			p := joinPointer(ptr, k)
			if _, wasKnown := known[k]; !wasKnown {
				a.stats(variantKey{Path: p, Type: typeUndefined, Hint: ""}).LinesWith += linesCompletedBeforeCurrent
				if err := a.walkValue(p, k, childVal, linesCompletedBeforeCurrent); err != nil {
					return err
				}
				known[k] = struct{}{}
				continue
			}
			if err := a.walkValue(p, k, childVal, linesCompletedBeforeCurrent); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported JSON type %T", v)
	}
}

func (a *Accumulator) linesTotal() int {
	return a.successfulLines
}

func (a *Accumulator) variantsAt(path string) []variantKey {
	var keys []variantKey
	for k := range a.variants {
		if k.Path == path && a.variants[k].LinesWith > 0 {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Type != keys[j].Type {
			return keys[i].Type < keys[j].Type
		}
		return keys[i].Hint < keys[j].Hint
	})
	return keys
}

func (a *Accumulator) allPaths() []string {
	seen := make(map[string]struct{})
	for k, st := range a.variants {
		if st.LinesWith > 0 {
			seen[k.Path] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
