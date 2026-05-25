# Agent context: guesschema

Go module **`github.com/Olian04/guesschema`** ([repo](https://github.com/Olian04/guesschema)). CLI: **JSONL stdin → guessed JSON Schema 2020-12 stdout**; optional **`--debug`** on stderr. No HTTP server, no Prometheus, no YAML config in this repo shape.

## Layout

| Path | Role |
| --- | --- |
| `cmd/guesschema` | CLI: urfave/cli v3, signals, flags → `internal/guesschema.New` + `Guesser.Run`. |
| `cmd/guesschema/version` | `Version`, `Revision`, `BuildTime` for `-ldflags -X` and `--version`. |
| `pkg/guesschema` | Public library: `New`, `Guesser`, `With*`, and domain (`Accumulator`, `BuildSchema`, …) → `internal/guesschema`. |
| `internal/guesschema` | Orchestration (JSONL read, emit, `Guesser`) plus domain (accumulator, materialize, RFC 6901 pointers). |
| `internal/hints` | Property key + value heuristics → JSON Schema `format` annotations (not tested directly). |
| `test/` | **All** tests: unit (`test/unit/...`) and blackbox (`test/blackbox/...`). No `*_test.go` under `cmd/`, `internal/`, or `pkg/`. |
| `test/unit/pkg/guesschema/...` | Library contract: import `pkg/guesschema` only (`package guesschema_test`). |
| `test/blackbox/...` | CLI contract: build `cmd/guesschema`, run via `exec` (shell-style); no imports of `pkg` or `internal`. |

## Testing (core rule)

**Every test file lives under `test/`.** Production trees (`cmd/`, `internal/`, `pkg/`) contain no tests.

**`internal/guesschema` is not a test target.** Tests guard consumer contracts only:

1. **Library** — `test/unit/pkg/guesschema/`: public `pkg/guesschema` API.
2. **CLI** — `test/blackbox/`: compiled binary (JSONL I/O, flags, exit codes). Invalid flags are covered here, not in `cmd/`.

If behavior is untestable through `pkg` or the CLI, extend the public API or CLI surface rather than adding tests under `internal/` or beside `cmd/`.

## Dependency direction

`cmd/guesschema` → `internal/guesschema` · `pkg/guesschema` → `internal/guesschema` · `internal/guesschema` → `internal/hints`. **`internal` must not import `pkg`.**

## Behavior summary

- **`--no-extra`:** omit vendor extensions by stripping JSON object keys starting with **`x-`** before stdout (recursive).
- **Single emit:** read at most **`--read-window`** (default **1s**), emit one schema, exit; EOF before budget → emit on EOF.
- **Interrupt:** SIGINT/SIGTERM (or canceled **`Run` ctx**) stops reading and still emits one schema from lines observed so far; emit checks **`ctx`** during **`buildSchema`** so Ctrl+C during a long materialize can abort promptly; CLI exits 0 after successful emit.
- **Materialize:** **`buildSchema`** builds a path index once per emit (avoids scanning all variants on every nested property).
- **Library `New`:** defaults only in `defaultConfig()`; `validate()` rejects invalid overrides; CLI flag checks live in `cmd/guesschema/flags.go`.

## Errors and logging

Use `%w` when wrapping errors in `cmd` / `internal/guesschema`. CLI **`--debug`** builds a stderr slog logger passed via **`WithLogger`**; library default is discard. **`Guesser`** is safe for concurrent **`Run`**; per-run state stays local to each call.

## Commands (`make`)

`build` → `./dist/guesschema`, `run`, `test`, `test-race`, `lint`, `format`.

## Releases

GoReleaser publishes **pre-built binaries** and SBOMs to GitHub Releases (no Docker image). Install from source with **`go install github.com/Olian04/guesschema/cmd/guesschema@<tag>`**; in **another** module, **`go get -tool`** + **`go tool guesschema`** pins the CLI. In **this** repo, use **`go run ./cmd/guesschema`** or **`make build`** — do not list `cmd/guesschema` under **`tool`** in `go.mod` (that block is for external dev tools like golangci-lint).

## Go proverbs ([source](https://go-proverbs.github.io/), compressed)

Concurrency channels coordinate mutex serializes · not parallelism · small interface sharp · zero value useful · `any` untyped tame it · gofmt settles bikeshed · tiny copy beats dep hairball syscall/cgo build tags isolate · cgo not Go · `unsafe` no contract · clarity beats wit · reflection stay cold path · errors values inspect wrap once · architecture name docs users · panic stays in `main` / hard startup.

## Uber style distill ([guide](https://github.com/uber-go/guide/blob/master/style.md), compressed)

Rare `*Iface` · `var _ I = (*T)(nil)` at export boundary · defer unlock pairs · chan buffer zero or one usually · slice/map copy exported API boundaries · typed errors `%w` chain handle once · assert comma-ok · goroutine bounded ctx/waitgroup · no zombie `init()` · globals inject not mutate · exits from `main` only · strconv hot paths · structs field-named literals · table tests sub `t.Run`.

Canon links above beat bullet memory when tradeoff unclear.
