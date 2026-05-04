package schemaguess

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
)

// Accumulator holds cross-line schema guess state for one read window.
type Accumulator struct {
	SuccessfulLines int
	InvalidJSON     int
	Variants        map[VariantKey]*VariantStats
	knownKeys       map[string]map[string]struct{} // pointer P -> child keys ever seen
}

// NewAccumulator returns an empty accumulator.
func NewAccumulator() *Accumulator {
	return &Accumulator{
		Variants:  make(map[VariantKey]*VariantStats),
		knownKeys: make(map[string]map[string]struct{}),
	}
}

func (a *Accumulator) stats(key VariantKey) *VariantStats {
	s := a.Variants[key]
	if s == nil {
		s = &VariantStats{}
		a.Variants[key] = s
	}
	return s
}

// Reset clears all state for a new window.
func (a *Accumulator) Reset() {
	a.SuccessfulLines = 0
	a.InvalidJSON = 0
	clear(a.Variants)
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
		a.InvalidJSON++
		return nil
	}
	linesBefore := a.SuccessfulLines
	if err := a.walkValue("", v, linesBefore); err != nil {
		return err
	}
	a.SuccessfulLines++
	return nil
}

func trimLine(b []byte) []byte {
	// trim ASCII space only; JSONL often has no trailing newline in buffer
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

func (a *Accumulator) walkValue(ptr string, v any, linesCompletedBeforeCurrent int) error {
	switch val := v.(type) {
	case nil:
		a.stats(VariantKey{Path: ptr, Type: TypeNull, Hint: ""}).LinesWith++
		return nil
	case bool:
		a.stats(VariantKey{Path: ptr, Type: TypeBoolean, Hint: ""}).LinesWith++
		return nil
	case float64:
		if math.IsInf(val, 0) || math.IsNaN(val) {
			return fmt.Errorf("non-finite number at %q", ptr)
		}
		a.stats(VariantKey{Path: ptr, Type: TypeNumber, Hint: ""}).LinesWith++
		return nil
	case string:
		hint := stringHint(val)
		a.stats(VariantKey{Path: ptr, Type: TypeString, Hint: hint}).LinesWith++
		return nil
	case []any:
		a.stats(VariantKey{Path: ptr, Type: TypeArray, Hint: ""}).LinesWith++
		for i, el := range val {
			child := JoinPointer(ptr, fmtInt(i))
			if err := a.walkValue(child, el, linesCompletedBeforeCurrent); err != nil {
				return err
			}
		}
		return nil
	case map[string]any:
		a.stats(VariantKey{Path: ptr, Type: TypeObject, Hint: ""}).LinesWith++
		known := a.ensureKnownKeys(ptr)
		present := make(map[string]struct{}, len(val))
		for k := range val {
			present[k] = struct{}{}
		}
		// 1. known \ present -> undefined +1 each
		for k := range known {
			if _, ok := present[k]; !ok {
				p := JoinPointer(ptr, k)
				a.stats(VariantKey{Path: p, Type: TypeUndefined, Hint: ""}).LinesWith++
			}
		}
		// 2 & 3. each present key: first sight vs known
		for k, childVal := range val {
			p := JoinPointer(ptr, k)
			if _, wasKnown := known[k]; !wasKnown {
				a.stats(VariantKey{Path: p, Type: TypeUndefined, Hint: ""}).LinesWith += linesCompletedBeforeCurrent
				if err := a.walkValue(p, childVal, linesCompletedBeforeCurrent); err != nil {
					return err
				}
				known[k] = struct{}{}
				continue
			}
			if err := a.walkValue(p, childVal, linesCompletedBeforeCurrent); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported JSON type %T", v)
	}
}

func fmtInt(i int) string {
	// small fast path
	return fmt.Sprintf("%d", i)
}

func stringHint(s string) string {
	if len(s) > 0 && (s[0] == '2' || s[0] == '1') && len(s) >= 10 {
		// very light RFC3339-ish signal for format: date-time
		for _, r := range s {
			if r == 'T' || r == 't' {
				return "date-time"
			}
		}
	}
	return ""
}

// LinesTotal returns successful line count for stats siblings.
func (a *Accumulator) LinesTotal() int {
	return a.SuccessfulLines
}

// Likelihood returns lines_with / lines_total for the window.
func Likelihood(linesWith, linesTotal int) float64 {
	if linesTotal <= 0 {
		return 0
	}
	return float64(linesWith) / float64(linesTotal)
}

// VariantsAt groups variant keys that share the same JSON Pointer path (different type/hint).
func (a *Accumulator) VariantsAt(path string) []VariantKey {
	var keys []VariantKey
	for k := range a.Variants {
		if k.Path == path && a.Variants[k].LinesWith > 0 {
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

// AllPaths returns sorted unique paths that have any non-zero variant.
func (a *Accumulator) AllPaths() []string {
	seen := make(map[string]struct{})
	for k, st := range a.Variants {
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
