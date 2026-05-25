package guesschema

const (
	typeUndefined = "undefined"
	typeNull      = "null"
	typeBoolean   = "boolean"
	typeNumber    = "number"
	typeString    = "string"
	typeObject    = "object"
	typeArray     = "array"
)

type variantKey struct {
	Path string
	Type string
	Hint string
}

type variantStats struct {
	LinesWith int
}

func (s *variantStats) likelihood(linesTotal int) float64 {
	if linesTotal <= 0 {
		return 0
	}
	return float64(s.LinesWith) / float64(linesTotal)
}
