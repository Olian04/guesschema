package guesschema

import (
	"sort"
	"strconv"
)

// materializeIndex is built once per BuildSchema from the variant map.
type materializeIndex struct {
	allPaths       []string
	variantsByPath map[string][]variantKey
	linesAtPath    map[string]int
	children       map[string][]string
	arrayIndicesByPath map[string][]string
}

func newMaterializeIndex(acc *Accumulator) *materializeIndex {
	byPath := make(map[string][]variantKey)
	linesAt := make(map[string]int)
	pathSet := make(map[string]struct{})

	for k, st := range acc.variants {
		if st.LinesWith <= 0 {
			continue
		}
		pathSet[k.Path] = struct{}{}
		if k.Type != typeUndefined {
			linesAt[k.Path] += st.LinesWith
		}
		byPath[k.Path] = append(byPath[k.Path], k)
	}

	for p, keys := range byPath {
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].Type != keys[j].Type {
				return keys[i].Type < keys[j].Type
			}
			return keys[i].Hint < keys[j].Hint
		})
		byPath[p] = keys
	}

	allPaths := make([]string, 0, len(pathSet))
	for p := range pathSet {
		allPaths = append(allPaths, p)
	}
	sort.Strings(allPaths)

	childSeen := make(map[string]map[string]struct{})
	arrSeen := make(map[string]map[string]struct{})

	for _, p := range allPaths {
		segs := splitPointer(p)
		if len(segs) == 0 {
			continue
		}
		parentPtr := ""
		for i := 0; i < len(segs)-1; i++ {
			parentPtr = joinPointer(parentPtr, segs[i])
		}
		child := segs[len(segs)-1]
		if childSeen[parentPtr] == nil {
			childSeen[parentPtr] = make(map[string]struct{})
		}
		childSeen[parentPtr][child] = struct{}{}
		if _, err := strconv.Atoi(child); err == nil {
			if arrSeen[parentPtr] == nil {
				arrSeen[parentPtr] = make(map[string]struct{})
			}
			arrSeen[parentPtr][child] = struct{}{}
		}
	}

	children := make(map[string][]string, len(childSeen))
	for parent, set := range childSeen {
		names := make([]string, 0, len(set))
		for name := range set {
			names = append(names, name)
		}
		sort.Strings(names)
		children[parent] = names
	}

	arrayIx := make(map[string][]string, len(arrSeen))
	for parent, set := range arrSeen {
		var indices []string
		for ix := range set {
			indices = append(indices, ix)
		}
		sort.Slice(indices, func(i, j int) bool {
			ai, _ := strconv.Atoi(indices[i])
			aj, _ := strconv.Atoi(indices[j])
			return ai < aj
		})
		arrayIx[parent] = indices
	}

	return &materializeIndex{
		allPaths:       allPaths,
		variantsByPath: byPath,
		linesAtPath:    linesAt,
		children:       children,
		arrayIndicesByPath: arrayIx,
	}
}

func (idx *materializeIndex) concreteVariants(path string) []variantKey {
	keys := idx.variantsByPath[path]
	out := make([]variantKey, 0, len(keys))
	for _, k := range keys {
		if k.Type == typeUndefined {
			continue
		}
		out = append(out, k)
	}
	return out
}

func (idx *materializeIndex) concreteLinesAtPath(path string) int {
	return idx.linesAtPath[path]
}

func (idx *materializeIndex) directChildKeys(ptr string) []string {
	return idx.children[ptr]
}

func (idx *materializeIndex) arrayIndices(ptr string) []string {
	return idx.arrayIndicesByPath[ptr]
}
