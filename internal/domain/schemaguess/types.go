package schemaguess

// ValueType names materialized JSON Schema primary types (plus undefined for absence).
const (
	TypeUndefined = "undefined"
	TypeNull      = "null"
	TypeBoolean   = "boolean"
	TypeNumber    = "number"
	TypeString    = "string"
	TypeObject    = "object"
	TypeArray     = "array"
)

// VariantKey identifies one accumulator row: path + value type + optional hint.
type VariantKey struct {
	Path string
	Type string
	Hint string // semantic hint (e.g. format name) or empty
}

// VariantStats holds per-row counters for materialization.
type VariantStats struct {
	LinesWith int
}
