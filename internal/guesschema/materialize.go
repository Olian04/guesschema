package guesschema

import (
	"sort"
	"strconv"
	"time"
)

// BuildSchema returns a JSON Schema 2020-12 root document map (encoding/json compatible).
func BuildSchema(acc *Accumulator, variantThreshold float64, generatedAt time.Time) map[string]any {
	lt := acc.linesTotal()
	root := map[string]any{
		"$schema":                         "https://json-schema.org/draft/2020-12/schema",
		"x-guesschema-generated-at":       generatedAt.UTC().Format(time.RFC3339Nano),
		"x-guesschema-invalid-json-lines": acc.invalidJSON,
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
		return map[string]any{"type": "object"}
	}
	if useOneOf(variants, acc, linesTotal, threshold) {
		return oneOfSchema(acc, ptr, variants, linesTotal, threshold)
	}
	w := pickWinner(variants, acc, linesTotal)
	return singleBranchSchema(acc, ptr, w, linesTotal, threshold)
}

func concreteVariants(acc *Accumulator, ptr string) []variantKey {
	keys := acc.variantsAt(ptr)
	var out []variantKey
	for _, k := range keys {
		if k.Type == typeUndefined {
			continue
		}
		if acc.variants[k].LinesWith == 0 {
			continue
		}
		out = append(out, k)
	}
	return out
}

func concreteLinesAtPath(acc *Accumulator, path string) int {
	n := 0
	for k, st := range acc.variants {
		if k.Path != path || k.Type == typeUndefined {
			continue
		}
		n += st.LinesWith
	}
	return n
}

func propertyLikelihoodOne(acc *Accumulator, path string, linesTotal int) bool {
	if linesTotal <= 0 {
		return false
	}
	return concreteLinesAtPath(acc, path) == linesTotal
}

func useOneOf(keys []variantKey, acc *Accumulator, linesTotal int, threshold float64) bool {
	if len(keys) <= 1 {
		return false
	}
	for _, k := range keys {
		st := acc.variants[k]
		if !(st.likelihood(linesTotal) > threshold) {
			return false
		}
	}
	return true
}

func pickWinner(keys []variantKey, acc *Accumulator, linesTotal int) variantKey {
	var best variantKey
	bestScore := -1
	bestL := -1.0
	bestLex := ""
	first := true
	for _, k := range keys {
		st := acc.variants[k]
		lik := st.likelihood(linesTotal)
		lex := k.Type + "\x00" + k.Hint
		if first || st.LinesWith > bestScore ||
			(st.LinesWith == bestScore && lik > bestL) ||
			(st.LinesWith == bestScore && lik == bestL && lex < bestLex) {
			best = k
			bestScore = st.LinesWith
			bestL = lik
			bestLex = lex
			first = false
		}
	}
	return best
}

func statsObject(st *variantStats, linesTotal int) map[string]any {
	return map[string]any{
		"x-guesschema-lines-with":  st.LinesWith,
		"x-guesschema-lines-total": linesTotal,
		"x-guesschema-likelihood":  st.likelihood(linesTotal),
	}
}

func mergeStats(base map[string]any, st *variantStats, linesTotal int) {
	for k, v := range statsObject(st, linesTotal) {
		base[k] = v
	}
}

func singleBranchSchema(acc *Accumulator, ptr string, k variantKey, linesTotal int, threshold float64) map[string]any {
	st := acc.variants[k]
	switch k.Type {
	case typeObject:
		return objectSchema(acc, ptr, linesTotal, threshold, st)
	case typeArray:
		return arraySchema(acc, ptr, linesTotal, threshold, st)
	default:
		m := leafTypeMap(k)
		mergeStats(m, st, linesTotal)
		return m
	}
}

func oneOfSchema(acc *Accumulator, ptr string, keys []variantKey, linesTotal int, threshold float64) map[string]any {
	sort.Slice(keys, func(i, j int) bool {
		ki, kj := keys[i], keys[j]
		if ki.Type != kj.Type {
			return ki.Type < kj.Type
		}
		return ki.Hint < kj.Hint
	})
	branches := make([]any, 0, len(keys))
	for _, k := range keys {
		st := acc.variants[k]
		var br map[string]any
		switch k.Type {
		case typeObject:
			br = objectSchema(acc, ptr, linesTotal, threshold, st)
		case typeArray:
			br = arraySchema(acc, ptr, linesTotal, threshold, st)
		default:
			br = leafTypeMap(k)
			mergeStats(br, st, linesTotal)
		}
		branches = append(branches, br)
	}
	return map[string]any{"oneOf": branches}
}

func leafTypeMap(k variantKey) map[string]any {
	m := map[string]any{"type": k.Type}
	if k.Hint == "" {
		return m
	}
	switch k.Type {
	case typeString, typeNumber:
		m["format"] = k.Hint
	default:
		m["x-guesschema-hint"] = k.Hint
	}
	return m
}

func objectSchema(acc *Accumulator, ptr string, linesTotal int, threshold float64, objectSt *variantStats) map[string]any {
	props := make(map[string]any)
	required := make([]string, 0)

	childNames := directChildKeys(ptr, acc.allPaths())
	var sorted []string
	for n := range childNames {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)

	for _, name := range sorted {
		cp := joinPointer(ptr, name)
		sub := materializeAt(acc, cp, threshold, linesTotal)
		props[name] = sub
		if propertyLikelihoodOne(acc, cp, linesTotal) {
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
	mergeStats(out, objectSt, linesTotal)
	return out
}

func arraySchema(acc *Accumulator, ptr string, linesTotal int, threshold float64, arraySt *variantStats) map[string]any {
	indices := arrayIndices(ptr, acc.allPaths())
	var itemsSchema map[string]any
	if len(indices) == 0 {
		itemsSchema = map[string]any{"type": "object"}
	} else if len(indices) == 1 {
		itemsSchema = materializeAt(acc, joinPointer(ptr, indices[0]), threshold, linesTotal)
	} else {
		branches := make([]any, 0, len(indices))
		for _, ix := range indices {
			branches = append(branches, materializeAt(acc, joinPointer(ptr, ix), threshold, linesTotal))
		}
		itemsSchema = map[string]any{"oneOf": branches}
	}
	out := map[string]any{
		"type":  "array",
		"items": itemsSchema,
	}
	mergeStats(out, arraySt, linesTotal)
	return out
}

func directChildKeys(ptr string, allPaths []string) map[string]struct{} {
	parentSegs := splitPointer(ptr)
	out := make(map[string]struct{})
	for _, p := range allPaths {
		segs := splitPointer(p)
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
	parentSegs := splitPointer(ptr)
	seen := make(map[string]struct{})
	for _, p := range allPaths {
		segs := splitPointer(p)
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
