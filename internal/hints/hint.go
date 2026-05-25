// Package hints infers JSON Schema format annotations from JSON property keys and values.
package hints

// For returns a JSON Schema-oriented annotation for a property key and decoded JSON value, or "" if none apply.
// Dispatch is by JSON value type; key-based heuristics supplement value-based ones.
func For(propKey string, v any) string {
	switch val := v.(type) {
	case string:
		return stringFor(propKey, val)
	case float64:
		return numberFor(propKey, val)
	case bool:
		return boolFor(propKey)
	case nil:
		return keyFor(propKey)
	default:
		return ""
	}
}
