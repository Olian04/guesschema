package guesschema

// Guesser holds frozen configuration for inferring JSON Schema from JSONL.
// It has no per-run state and is safe for concurrent Run calls from multiple
// goroutines. Do not share the same io.Reader across concurrent Run calls.
type Guesser struct {
	cfg config
}

// New applies opts, validates, and returns a Guesser safe for concurrent use.
func New(opts ...Option) (*Guesser, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &Guesser{cfg: cfg}, nil
}
