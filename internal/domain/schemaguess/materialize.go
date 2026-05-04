package schemaguess

import (
	"sort"
	"strconv"
	"time"
)

// BuildSchema returns a JSON Schema 2020-12 root document map (encoding/json compatible).
func BuildSchema(acc *Accumulator, variantThreshold float64, generatedAt time.Time) map[string]any {
	lt := acc.LinesTotal()
	root := map[string]any{
		"$schema":                         "https://json-schema.org/draft/2020-12/schema",
		"x-guesschema-generated-at":       generatedAt.UTC().Format(time.RFC3339Nano),
		"x-guesschema-invalid-json-lines": acc.InvalidJSON,
	}
	body := materializeAt(acc, "", variantThreshold, lt)
	for k, v := range body {
		root[k] = v
	}
	return root
}

func materializeAt(acc *Accumulator, ptr string, threshold float64, linesTotal int) map[string]any {
	variants := concreteVariants(acc, ptr)
	if len(variants) == 0 {
		// no observations at this node (should be rare)
		return map[string]any{"type": "object"}
	}
	if useOneOf(variants, acc, linesTotal, threshold) {
		return oneOfSchema(acc, ptr, variants, linesTotal, threshold)
	}
	w := pickWinner(variants, acc, linesTotal)
	return singleBranchSchema(acc, ptr, w, linesTotal, threshold)
}

func concreteVariants(acc *Accumulator, ptr string) []VariantKey {
	keys := acc.VariantsAt(ptr)
	var out []VariantKey
	for _, k := range keys {
		if k.Type == TypeUndefined {
			continue
		}
		if acc.Variants[k].LinesWith == 0 {
			continue
		}
		out = append(out, k)
	}
	return out
}

func useOneOf(keys []VariantKey, acc *Accumulator, linesTotal int, threshold float64) bool {
	if len(keys) <= 1 {
		return false
	}
	for _, k := range keys {
		lw := acc.Variants[k].LinesWith
		if !(Likelihood(lw, linesTotal) > threshold) {
			return false
		}
	}
	return true
}

func pickWinner(keys []VariantKey, acc *Accumulator, linesTotal int) VariantKey {
	var best VariantKey
	bestScore := -1
	bestL := -1.0
	bestLex := ""
	first := true
	for _, k := range keys {
		lw := acc.Variants[k].LinesWith
		lik := Likelihood(lw, linesTotal)
		lex := k.Type + "\x00" + k.Hint
		if first || lw > bestScore ||
			(lw == bestScore && lik > bestL) ||
			(lw == bestScore && lik == bestL && lex < bestLex) {
			best = k
			bestScore = lw
			bestL = lik
			bestLex = lex
			first = false
		}
	}
	return best
}

func statsObject(linesWith, linesTotal int) map[string]any {
	return map[string]any{
		"x-guesschema-lines-with":  linesWith,
		"x-guesschema-lines-total": linesTotal,
		"x-guesschema-likelihood":  Likelihood(linesWith, linesTotal),
	}
}

func mergeStats(base map[string]any, linesWith, linesTotal int) {
	for k, v := range statsObject(linesWith, linesTotal) {
		base[k] = v
	}
}

func singleBranchSchema(acc *Accumulator, ptr string, k VariantKey, linesTotal int, threshold float64) map[string]any {
	lw := acc.Variants[k].LinesWith
	switch k.Type {
	case TypeObject:
		return objectSchema(acc, ptr, linesTotal, threshold, lw)
	case TypeArray:
		return arraySchema(acc, ptr, linesTotal, threshold, lw)
	default:
		m := leafTypeMap(k)
		mergeStats(m, lw, linesTotal)
		return m
	}
}

func oneOfSchema(acc *Accumulator, ptr string, keys []VariantKey, linesTotal int, threshold float64) map[string]any {
	sort.Slice(keys, func(i, j int) bool {
		ki, kj := keys[i], keys[j]
		if ki.Type != kj.Type {
			return ki.Type < kj.Type
		}
		return ki.Hint < kj.Hint
	})
	branches := make([]any, 0, len(keys))
	for _, k := range keys {
		lw := acc.Variants[k].LinesWith
		var br map[string]any
		switch k.Type {
		case TypeObject:
			br = objectSchema(acc, ptr, linesTotal, threshold, lw)
		case TypeArray:
			br = arraySchema(acc, ptr, linesTotal, threshold, lw)
		default:
			br = leafTypeMap(k)
			mergeStats(br, lw, linesTotal)
		}
		branches = append(branches, br)
	}
	return map[string]any{"oneOf": branches}
}

func leafTypeMap(k VariantKey) map[string]any {
	m := map[string]any{"type": k.Type}
	if k.Type == TypeString && k.Hint != "" {
		m["format"] = k.Hint
	}
	return m
}

func objectSchema(acc *Accumulator, ptr string, linesTotal int, threshold float64, objectLinesWith int) map[string]any {
	props := make(map[string]any)
	required := make([]string, 0)

	childNames := directChildKeys(ptr, acc.AllPaths())
	var sorted []string
	for n := range childNames {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)

	for _, name := range sorted {
		cp := JoinPointer(ptr, name)
		u := acc.Variants[VariantKey{Path: cp, Type: TypeUndefined, Hint: ""}]
		undef := 0
		if u != nil {
			undef = u.LinesWith
		}
		sub := materializeAt(acc, cp, threshold, linesTotal)
		props[name] = sub
		if undef == 0 && linesTotal > 0 {
			required = append(required, name)
		}
	}

	out := map[string]any{
		"type":                 "object",
		"properties":           props,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		sort.Strings(required)
		out["required"] = required
	}
	mergeStats(out, objectLinesWith, linesTotal)
	return out
}

func arraySchema(acc *Accumulator, ptr string, linesTotal int, threshold float64, arrayLinesWith int) map[string]any {
	indices := arrayIndices(ptr, acc.AllPaths())
	var itemsSchema map[string]any
	if len(indices) == 0 {
		itemsSchema = map[string]any{"type": "object"}
	} else if len(indices) == 1 {
		itemsSchema = materializeAt(acc, JoinPointer(ptr, indices[0]), threshold, linesTotal)
	} else {
		branches := make([]any, 0, len(indices))
		for _, ix := range indices {
			branches = append(branches, materializeAt(acc, JoinPointer(ptr, ix), threshold, linesTotal))
		}
		itemsSchema = map[string]any{"oneOf": branches}
	}
	out := map[string]any{
		"type":  "array",
		"items": itemsSchema,
	}
	mergeStats(out, arrayLinesWith, linesTotal)
	return out
}

func directChildKeys(ptr string, allPaths []string) map[string]struct{} {
	parentSegs := SplitPointer(ptr)
	out := make(map[string]struct{})
	for _, p := range allPaths {
		segs := SplitPointer(p)
		if len(segs) != len(parentSegs)+1 {
			continue
		}
		if !hasPrefixSegs(segs, parentSegs) {
			continue
		}
		out[segs[len(parentSegs)]] = struct{}{}
	}
	return out
}

func hasPrefixSegs(full, prefix []string) bool {
	if len(prefix) > len(full) {
		return false
	}
	for i := range prefix {
		if full[i] != prefix[i] {
			return false
		}
	}
	return true
}

func arrayIndices(ptr string, allPaths []string) []string {
	parentSegs := SplitPointer(ptr)
	seen := make(map[string]struct{})
	for _, p := range allPaths {
		segs := SplitPointer(p)
		if len(segs) != len(parentSegs)+1 {
			continue
		}
		if !hasPrefixSegs(segs, parentSegs) {
			continue
		}
		tok := segs[len(parentSegs)]
		if _, err := strconv.Atoi(tok); err == nil {
			seen[tok] = struct{}{}
		}
	}
	var out []string
	for ix := range seen {
		out = append(out, ix)
	}
	sort.Slice(out, func(i, j int) bool {
		ai, _ := strconv.Atoi(out[i])
		aj, _ := strconv.Atoi(out[j])
		return ai < aj
	})
	return out
}
