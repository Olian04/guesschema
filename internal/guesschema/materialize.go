package guesschema

import (
	"context"
	"sort"
	"time"
)

// BuildSchema returns a JSON Schema 2020-12 root document map (encoding/json compatible).
func BuildSchema(acc *Accumulator, variantThreshold float64, generatedAt time.Time) map[string]any {
	doc, _ := buildSchema(context.Background(), acc, variantThreshold, generatedAt)
	return doc
}

func buildSchema(ctx context.Context, acc *Accumulator, variantThreshold float64, generatedAt time.Time) (map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	lt := acc.linesTotal()
	idx := newMaterializeIndex(acc)
	root := map[string]any{
		"$schema":                         "https://json-schema.org/draft/2020-12/schema",
		"x-guesschema-generated-at":       generatedAt.UTC().Format(time.RFC3339Nano),
		"x-guesschema-invalid-json-lines": acc.invalidJSON,
	}
	body, err := materializeAt(ctx, idx, acc, "", variantThreshold, lt)
	if err != nil {
		return nil, err
	}
	for k, v := range body {
		root[k] = v
	}
	return root, nil
}

func materializeAt(ctx context.Context, idx *materializeIndex, acc *Accumulator, ptr string, threshold float64, linesTotal int) (map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	variants := idx.concreteVariants(ptr)
	if len(variants) == 0 {
		return map[string]any{"type": "object"}, nil
	}
	if useOneOf(variants, acc, linesTotal, threshold) {
		return oneOfSchema(ctx, idx, acc, ptr, variants, linesTotal, threshold)
	}
	w := pickWinner(variants, acc, linesTotal)
	return singleBranchSchema(ctx, idx, acc, ptr, w, linesTotal, threshold)
}

func propertyLikelihoodOne(idx *materializeIndex, path string, linesTotal int) bool {
	if linesTotal <= 0 {
		return false
	}
	return idx.concreteLinesAtPath(path) == linesTotal
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

func singleBranchSchema(ctx context.Context, idx *materializeIndex, acc *Accumulator, ptr string, k variantKey, linesTotal int, threshold float64) (map[string]any, error) {
	st := acc.variants[k]
	switch k.Type {
	case typeObject:
		return objectSchema(ctx, idx, acc, ptr, linesTotal, threshold, st)
	case typeArray:
		return arraySchema(ctx, idx, acc, ptr, linesTotal, threshold, st)
	default:
		m := leafTypeMap(k)
		mergeStats(m, st, linesTotal)
		return m, nil
	}
}

func oneOfSchema(ctx context.Context, idx *materializeIndex, acc *Accumulator, ptr string, keys []variantKey, linesTotal int, threshold float64) (map[string]any, error) {
	sort.Slice(keys, func(i, j int) bool {
		ki, kj := keys[i], keys[j]
		if ki.Type != kj.Type {
			return ki.Type < kj.Type
		}
		return ki.Hint < kj.Hint
	})
	branches := make([]any, 0, len(keys))
	for _, k := range keys {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		st := acc.variants[k]
		var br map[string]any
		var err error
		switch k.Type {
		case typeObject:
			br, err = objectSchema(ctx, idx, acc, ptr, linesTotal, threshold, st)
		case typeArray:
			br, err = arraySchema(ctx, idx, acc, ptr, linesTotal, threshold, st)
		default:
			br = leafTypeMap(k)
			mergeStats(br, st, linesTotal)
		}
		if err != nil {
			return nil, err
		}
		branches = append(branches, br)
	}
	return map[string]any{"oneOf": branches}, nil
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

func objectSchema(ctx context.Context, idx *materializeIndex, acc *Accumulator, ptr string, linesTotal int, threshold float64, objectSt *variantStats) (map[string]any, error) {
	props := make(map[string]any)
	required := make([]string, 0)

	for _, name := range idx.directChildKeys(ptr) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		cp := joinPointer(ptr, name)
		sub, err := materializeAt(ctx, idx, acc, cp, threshold, linesTotal)
		if err != nil {
			return nil, err
		}
		props[name] = sub
		if propertyLikelihoodOne(idx, cp, linesTotal) {
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
	return out, nil
}

func arraySchema(ctx context.Context, idx *materializeIndex, acc *Accumulator, ptr string, linesTotal int, threshold float64, arraySt *variantStats) (map[string]any, error) {
	indices := idx.arrayIndices(ptr)
	var itemsSchema map[string]any
	var err error
	if len(indices) == 0 {
		itemsSchema = map[string]any{"type": "object"}
	} else if len(indices) == 1 {
		itemsSchema, err = materializeAt(ctx, idx, acc, joinPointer(ptr, indices[0]), threshold, linesTotal)
	} else {
		branches := make([]any, 0, len(indices))
		for _, ix := range indices {
			var br map[string]any
			br, err = materializeAt(ctx, idx, acc, joinPointer(ptr, ix), threshold, linesTotal)
			if err != nil {
				return nil, err
			}
			branches = append(branches, br)
		}
		itemsSchema = map[string]any{"oneOf": branches}
	}
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"type":  "array",
		"items": itemsSchema,
	}
	mergeStats(out, arraySt, linesTotal)
	return out, nil
}
